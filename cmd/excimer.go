package cmd

import (
	"github.com/gookit/color"
	"github.com/jonhadfield/findexec"
	"github.com/sandstorm/drydock/util"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"os/signal"
)

// phpExcimerInstallScript is running in the debugImage
//   - we mount the inner container to /container (should be based on some base "official" Docker PHP image)
//   - reload the config
func phpExcimerInstallScript(pid string) string {
	// fall back to xdebug 3.1.6 for PHP 7.4 if xdebug 3.2 (the newest version) did not work
	return mountSlashContainer + `
cat << EOF | chroot /container
	export HTTP_PROXY=""
	export HTTPS_PROXY=""
	pecl install excimer
EOF

if [ ! -d /container$PHP_INI_DIR ]; then
    echo "!!!! PHP_INI_DIR not set."
    echo "!!!! please set it as env in docker compose."
    exit 1
fi;

cat << EOF > /container$PHP_INI_DIR/conf.d/excimer.ini
auto_prepend_file=/app/tracing/auto_prepend_file.php

extension=excimer.so

EOF

mkdir -p /container/app/tracing
mkdir -p /container/app/tracing/_traces
chmod -R 777 /container/app/tracing

cat << 'EOF' > /container/app/tracing/auto_prepend_file.php
<?php

// !!!!!!!!!!!!! DO NOT DELETE THIS FILE !!!!!!!!!!!!!!!!!!!!
// This file is included via auto_prepend_file in php.ini - and it is used for
// profiling.
// !!!!!!!!!!!!! DO NOT DELETE THIS FILE !!!!!!!!!!!!!!!!!!!!

function startExcimer() {
    static $excimer;
    if (!class_exists(\ExcimerProfiler::class)) {
        // excimer.so profiling extension not loaded.
        return;
    }

    $excimer = new ExcimerProfiler();
    $excimer->setPeriod( 0.001 ); // 1ms
    $excimer->setEventType( EXCIMER_REAL ); // OR: EXCIMER_CPU, but does not work on FreeBSD.
    $excimer->start();
    register_shutdown_function( function () use ( $excimer ) {
        $excimer->stop();
        $data = $excimer->getLog()->formatCollapsed();
        file_put_contents('/app/tracing/_traces/' . getmypid(), $data, FILE_APPEND);
    } );
}

// HINT: to start PHP continuous profiling, comment-in the following line.
startExcimer();

EOF

echo "restarting php-fpm"
pkill -USR2 php-fpm
`
}

func phpXExcimerDeactivateScript() string {
	return mountSlashContainer + `
rm /container$PHP_INI_DIR/conf.d/excimer.ini
pkill -USR2 php-fpm
`
}

func buildExcimerCommand() *cobra.Command {
	var debugImage string = "nicolaka/netshoot"

	var command = &cobra.Command{
		Use:   "excimer [flags] SERVICE-or-CONTAINER",
		Short: "Install Excimer Sampling Continuous Profiler in the given container",
		Long: color.Sprintf(`Usage:	drydock excimer [flags] SERVICE-OR-CONTAINER

Run excimer Continuous Profiler in the given PHP Container, and reloads
the PHP Process such that the debugger is enabled.

<op=underscore;>Options:</>
      --debug-image          What debugger docker image to use for executing nsenter (and optionally the NFS webdav server).
                             By default, nicolaka/netshoot is used

<op=underscore;>Examples</>

<op=bold;>Run excimer in a running PHP container</>
	drydock excimer <op=italic;>myContainer</>

<op=bold;>Run excimer in a running docker-compose service</>
	drydock excimer <op=italic;>my-docker-compose-service</>

<op=underscore;>Background:</>

    This command installs the excimer PHP extension into an existing Docker container, even if the container is locked
    down to a non-root user. Additionally, we reload the PHP process by using kill -USR2.

    This command is using <op=italic;>nsenter</> wrapped in a privileged docker container to install the PHP extension
    inside a running container as root.

`),
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			//isOpen := isXdebugPortOpenInIde("127.0.0.1", "9003")

			color.Println("")
			color.Println("")
			color.Println("<green>=====================================</>")
			color.Println("<green>Installing excimer into the container</>")
			color.Println("<green>and reloading PHP</>")
			color.Println("<green>=====================================</>")
			color.Println("")

			dockerContainerIdentifier, err := util.TryGetDockerContainerNameFromDockerCompose(args[0])

			if err != nil {
				// could not identify the docker container identifier; e.g. no docker-compose used.
				dockerContainerIdentifier = args[0]
			} else {
				color.Printf("<green>docker compose service </><fg=green;op=bold;>%s</><fg=green> found, entering it.</>\n", args[0])
				color.Println("")
			}

			pid, err := util.GetRootPidForDockerContainer(dockerContainerIdentifier)

			if err != nil || pid == "0" {
				// container not running
				color.Printf("<red>FATAL: Container </><fg=red;op=bold;>%s</><fg=red> not running.</>\n", dockerContainerIdentifier)
				color.Println("")
				os.Exit(1)
			}

			fullContainerName, err := util.GetFullContainerName(dockerContainerIdentifier)
			if err != nil {
				color.Printf("<red>FATAL: Could not extract container name for container </><fg=red;op=bold;>%s</><fg=red> - THIS SHOULD NOT HAPPEN. Please file a bug report.</>\n", dockerContainerIdentifier)
				color.Println("")
				os.Exit(1)
			}

			dockerExecutablePathAndFilename := findexec.Find("docker", "")

			// we need to get the ENV of the original container to find the PHP_INI_DIR (needed such that "docker-php-ext-enable" will work: https://github.com/docker-library/php/blob/67c242cb1529c70a3969a373ab333c53001c95b8/8.2-rc/bullseye/cli/docker-php-ext-enable)
			extraDockerRunArgs, err := util.GetEnvCliCallsForDockerRunFromContainerMetadata(fullContainerName)
			if err != nil {
				log.Printf("FATAL: Could not extract env variables for container '%s': %s - THIS SHOULD NOT HAPPEN. Please file a bug report.\n", dockerContainerIdentifier, err)
				os.Exit(1)
			}

			// Install excimer
			dockerRunCommand := dockerRunNsenterCommand(fullContainerName, debugImage, pid, extraDockerRunArgs)
			dockerRunCommand = append(dockerRunCommand, "--net") // to download files now, we need to mount the network filesystem.
			dockerRunCommand = append(dockerRunCommand, "/bin/bash")
			dockerRunCommand = append(dockerRunCommand, "-c")
			dockerRunCommand = append(dockerRunCommand, phpExcimerInstallScript(pid))

			dockerRunC := exec.Command(dockerExecutablePathAndFilename, dockerRunCommand[1:]...)
			dockerRunC.Env = os.Environ()
			dockerRunC.Stdout = os.Stdout
			dockerRunC.Stderr = os.Stderr
			dockerRunC.Run()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)

			printExcimerUsage(fullContainerName)
			// wait for ctrl-c
			<-c
			color.Println("<fg=yellow>Ctrl-C pressed. Aborting...</>")

			color.Println("<green>=====================================</>")
			color.Printf("<green>Disabling Excimer</>\n")
			color.Println("<green>=====================================</>")
			color.Println("")
			// Removing XDebug
			// Install XDEBUG and prepare for NFS Server
			dockerRunCommand = dockerRunNsenterCommand(fullContainerName, debugImage, pid, extraDockerRunArgs)
			dockerRunCommand = append(dockerRunCommand, "/bin/bash")
			dockerRunCommand = append(dockerRunCommand, "-c")
			dockerRunCommand = append(dockerRunCommand, phpXExcimerDeactivateScript())

			dockerRunC = exec.Command(dockerExecutablePathAndFilename, dockerRunCommand[1:]...)
			dockerRunC.Env = os.Environ()
			dockerRunC.Stdout = os.Stdout
			dockerRunC.Stderr = os.Stderr
			dockerRunC.Run()

			color.Println("<green>=====================================</>")
			color.Printf("<green>All done!</>\n")
			color.Println("<green>=====================================</>")
			color.Println("")

		},
	}

	command.Flags().StringVarP(&debugImage, "debug-image", "", "nicolaka/netshoot", "What debugger docker image to use for executing nsenter. By default, gists/nfs-server is used")

	return command
}

func printExcimerUsage(fullContainerName string) {
	color.Println("")
	color.Println("")
	color.Println("<fg=green>=====================================</>")
	color.Printf("<fg=green;op=bold>Excimer fully set up for %s</>\n", fullContainerName)
	color.Println("")
	color.Println("<fg=green>=====================================</>")
	color.Println("")
	color.Println("<fg=yellow>To stop debugging, </><fg=yellow;op=bold>press Ctrl-C</>")
}
