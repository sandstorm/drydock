package cmd

import (
	"github.com/gookit/color"
	"github.com/jonhadfield/findexec"
	"github.com/sandstorm/drydock/util"
	"github.com/spf13/cobra"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

// phpXdebugInstallScript is running in the debugImage
//   - we mount the inner container to /container (should be based on some base "official" Docker PHP image)
//   - we php-spx via Git inside nicolaka/netshoot (because we cannot know if git is installed inside the container)
//   - then, we compile and install php-spx inside the container. This runs as root, because we use the "execroot" mechanics
//     (important for the `make install` step).
//   - reload the config
func phpXdebugInstallScript(pid string) string {
	// fall back to xdebug 3.1.6 for PHP 7.4 if xdebug 3.2 (the newest version) did not work
	return mountSlashContainer + `
cat << EOF | chroot /container
	export HTTP_PROXY=""
	export HTTPS_PROXY=""
	pecl install xdebug || pecl install xdebug-3.1.6
EOF

if [ ! -d /container$PHP_INI_DIR ]; then
    echo "!!!! PHP_INI_DIR not set."
    echo "!!!! please set it as env in docker compose."
    exit 1
fi;

cat << EOF > /container$PHP_INI_DIR/conf.d/xdebug.ini
zend_extension=xdebug.so

xdebug.mode = develop,debug
xdebug.client_host = host.docker.internal
xdebug.discover_client_host = true
xdebug.max_nesting_level = 2048
EOF

echo "restarting php-fpm"
pkill -USR2 php-fpm
`
}

func phpXdebugDeactivateScript() string {
	return mountSlashContainer + `
rm /container$PHP_INI_DIR/conf.d/xdebug.ini
pkill -USR2 php-fpm
`
}

func buildXdebugCommand() *cobra.Command {
	var debugImage string = "nicolaka/netshoot"

	var command = &cobra.Command{
		Use:   "xdebug [flags] SERVICE-or-CONTAINER",
		Short: "Run Xdebug in the given container",
		Long: color.Sprintf(`Usage:	drydock xdebug [flags] SERVICE-OR-CONTAINER

Run Xdebug https://xdebug.org in the given PHP Container, and reloads
the PHP Process such that the debugger is enabled.

<op=underscore;>Options:</>
      --debug-image          What debugger docker image to use for executing nsenter (and optionally the NFS webdav server).
                             By default, nicolaka/netshoot is used

<op=underscore;>Examples</>

<op=bold;>Run Xdebug in a running PHP container</>
	drydock xdebug <op=italic;>myContainer</>

<op=bold;>Run Xdebug in a running docker-compose service</>
	drydock xdebug <op=italic;>my-docker-compose-service</>

<op=underscore;>Background:</>

    This command installs the Xdebug PHP extension into an existing Docker container, even if the container is locked
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
			color.Println("<green>Installing Xdebug into the container</>")
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

			// Install XDEBUG
			dockerRunCommand := dockerRunNsenterCommand(fullContainerName, debugImage, pid, extraDockerRunArgs)
			dockerRunCommand = append(dockerRunCommand, "--net") // to download files now, we need to mount the network filesystem.
			dockerRunCommand = append(dockerRunCommand, "/bin/bash")
			dockerRunCommand = append(dockerRunCommand, "-c")
			dockerRunCommand = append(dockerRunCommand, phpXdebugInstallScript(pid))

			dockerRunC := exec.Command(dockerExecutablePathAndFilename, dockerRunCommand[1:]...)
			dockerRunC.Env = os.Environ()
			dockerRunC.Stdout = os.Stdout
			dockerRunC.Stderr = os.Stderr
			dockerRunC.Run()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)

			printXdebugUsage(fullContainerName)
			// wait for ctrl-c
			<-c
			color.Println("<fg=yellow>Ctrl-C pressed. Aborting...</>")

			color.Println("<green>=====================================</>")
			color.Printf("<green>Disabling Xdebug</>\n")
			color.Println("<green>=====================================</>")
			color.Println("")
			// Removing XDebug
			// Install XDEBUG and prepare for NFS Server
			dockerRunCommand = dockerRunNsenterCommand(fullContainerName, debugImage, pid, extraDockerRunArgs)
			dockerRunCommand = append(dockerRunCommand, "/bin/bash")
			dockerRunCommand = append(dockerRunCommand, "-c")
			dockerRunCommand = append(dockerRunCommand, phpXdebugDeactivateScript())

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

func printXdebugUsage(fullContainerName string) {
	color.Println("")
	color.Println("")
	color.Println("<fg=green>=====================================</>")
	color.Printf("<fg=green;op=bold>Xdebug fully set up for %s</>\n", fullContainerName)
	color.Println("")
	color.Println("<fg=green>~~ Debugging Web Requests ~~</>")
	color.Println("")
	color.Println("<fg=green>- </><fg=green;op=bold;>xdebug_break()</>")
	color.Println("<fg=green>  in PHP code to set a breakpoint (recommended for Neos/Flow)</>")
	color.Println("<fg=green>- http://your-url-here/</><fg=green;op=bold;>?XDEBUG_SESSION=1</>")
	color.Println("<fg=green>  in your HTTP request to debug a single request</>")
	color.Println("<fg=green>- http://your-url-here/</><fg=green;op=bold;>?XDEBUG_SESSION_START=1</>")
	color.Println("<fg=green>  in your HTTP request to debug all requests in this session</>")
	color.Println("<fg=green>  (stop with XDEBUG_SESSION_STOP=1)</>")
	color.Println("")
	color.Println("<fg=green>~~ Debugging CLI requests ~~</>")
	color.Println("")
	color.Println("<fg=green>- </><fg=green;op=bold;>XDEBUG_SESSION=1</><fg=green> php ...</>")
	color.Println("<fg=green>  for CLI step Debugging</>")
	color.Println("")
	color.Println("<fg=green>~~ Set up PHPStorm/IntelliJ ~~</>")
	color.Println("<fg=green>- </><fg=green;op=bold;>Run -> Start Listening for PHP Debug Connections</>")
	color.Println("<fg=green>  needs to be enabled; otherwise connection to the IDE does not work.</>")
	color.Println("<fg=green>- You need to set up </><fg=green;op=bold;>path mappings</><fg=green> correctly, otherwise you cannot navigate</>")
	color.Println("<fg=green>  to the files in the IDE when a breakpoint is hit. This can be done as follows:</>")
	color.Println("")
	color.Println("<fg=green>  When a breakpoint is hit:</>")
	color.Println("<fg=green>     Debug Panel -> Threads&Variables</>")
	color.Println("<fg=green>     -> </><fg=green;op=bold;>Click to set up path mappings</>")
	color.Println("<fg=green>  When the path mapping is wrongly configured and you need to correct it:</>")
	color.Println("<fg=green>     Settings</>")
	color.Println("<fg=green>     -> Languages&Frameworks -> PHP -> Server</>")
	color.Println("<fg=green>     -> (add server if needed)</>")
	color.Println("<fg=green>     -> </><fg=green;op=bold;>Use Path Mappings</>")
	color.Println("")
	color.Println("<fg=green>~~ Debugging Neos/Flow ~~</>")
	color.Println("<fg=green>For debugging Neos/Flow, run with </><fg=green;op=bold;>--mount app/Data/Temporary,app/Packages</><fg=green>, because this</>")
	color.Println("<fg=green>allows to edit all files in the IDE.</>")
	color.Println("")
	color.Println("<fg=green>Additionally, enable Power Save Mode in IntelliJ to stop reindexing.</>")
	color.Println("<fg=green>=====================================</>")
	color.Println("")
	color.Println("<fg=yellow>To stop debugging, </><fg=yellow;op=bold>press Ctrl-C</>")
}

func isXdebugPortOpenInIde(host string, port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		defer conn.Close()
		return true
	}
	return false
}
