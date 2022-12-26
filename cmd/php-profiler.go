package cmd

import (
	"github.com/gookit/color"
	"github.com/jonhadfield/findexec"
	"github.com/sandstorm/docker-execroot/util"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
)

// phpSpxInstallScript is running in the debugImage (by default nicolaka/netshoot).
//   - we mount the inner container to /container (should be based on some base "official" Docker PHP image)
//   - we php-spx via Git inside nicolaka/netshoot (because we cannot know if git is installed inside the container)
//   - then, we compile and install php-spx inside the container. This runs as root, because we use the "execroot" mechanics
//     (important for the `make install` step).
//   - reload the config
const phpSpxInstallScript = mountSlashContainer + `
git clone --branch release/latest https://github.com/NoiseByNorthwest/php-spx.git /container/php-spx

cat << EOF | chroot /container
	cd /php-spx
	phpize
	./configure
	make
	make install
EOF

cat << EOF > /container$PHP_INI_DIR/conf.d/spx.ini
extension=spx.so

spx.http_enabled=1
spx.http_key="dev"
spx.http_ip_whitelist="*"
EOF

pkill -USR2 php-fpm
`

func buildPhpProfilerCommand() *cobra.Command {
	var debugImage string = "nicolaka/netshoot"

	var phpProfilerCommand = &cobra.Command{
		Use:   "phpprofiler [flags] SERVICE-or-CONTAINER",
		Short: "Install SPX PHP-Profiler in the given container",
		Long: color.Sprintf(`Usage:	docker phpprofiler [flags] SERVICE-OR-CONTAINER

Install the SPX PHP-Profiler into the given PHP Container, and restarts the PHP Process.
`),
		Args: cobra.MinimumNArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			dockerContainerIdentifier, err := util.TryGetDockerContainerNameFromDockerCompose(args[0])

			if err != nil {
				// could not identify the docker container identifier; e.g. no docker-compose used.
				dockerContainerIdentifier = args[0]
			} else {
				log.Printf("docker compose service '%s' found, entering it.\n", args[0])
			}

			pid, err := util.GetRootPidForDockerContainer(dockerContainerIdentifier)

			if err != nil {
				// container not running
				log.Printf("FATAL: Container '%s' not running\n", dockerContainerIdentifier)
				os.Exit(1)
			}

			fullContainerName, err := util.GetFullContainerName(dockerContainerIdentifier)
			if err != nil {
				log.Printf("FATAL: Could not extract container name for container '%s' - THIS SHOULD NOT HAPPEN. Please file a bug report.\n", dockerContainerIdentifier)
				os.Exit(1)
			}

			dockerExecutablePathAndFilename := findexec.Find("docker", "")

			color.Println("")
			color.Println("")
			color.Println("<green>=====================================</>")
			color.Println("<green>Installing PHP-SPX into the container</>")
			color.Println("<green>and reloading PHP</>")
			color.Println("<green>=====================================</>")
			color.Println("")
			color.Println("")
			// we need to get the ENV of the original container to find the PHP_INI_DIR (needed such that "docker-php-ext-enable" will work: https://github.com/docker-library/php/blob/67c242cb1529c70a3969a373ab333c53001c95b8/8.2-rc/bullseye/cli/docker-php-ext-enable)
			envVars, err := util.GetEnvCliCallsForDockerRunFromContainerMetadata(fullContainerName)
			if err != nil {
				log.Printf("FATAL: Could not extract env variables for container '%s': %s - THIS SHOULD NOT HAPPEN. Please file a bug report.\n", dockerContainerIdentifier, err)
				os.Exit(1)
			}

			// Install PHP-SPX
			dockerRunCommand := basicDockerRunCommand(fullContainerName, debugImage, pid, envVars)
			dockerRunCommand = append(dockerRunCommand, "/bin/bash")
			dockerRunCommand = append(dockerRunCommand, "-c")
			dockerRunCommand = append(dockerRunCommand, phpSpxInstallScript)

			c := exec.Command(dockerExecutablePathAndFilename, dockerRunCommand[1:]...)
			c.Env = os.Environ()
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin
			c.Run()

			color.Println("")
			color.Println("")
			color.Println("<fg=green>=====================================</>")
			color.Printf("<fg=green;op=bold>Finished installing PHP-SPX into %s</>\n", fullContainerName)
			color.Println("")
			color.Println("<fg=green>SPX Profiler URL:</>")
			hostPorts, _ := util.GetHostPorts(fullContainerName)
			for _, hostPort := range hostPorts {
				color.Printf("  - <fg=green;op=bold;>http://127.0.0.1:%d/?SPX_UI_URI=/&SPX_KEY=dev</>\n", hostPort)
			}
			color.Println("")
			color.Println("<fg=green>Profiling CLI requests:</>")
			color.Println("<fg=green>- </><fg=green;op=bold;>SPX_ENABLED=1</><fg=green> php ...</>")
			color.Println("<fg=green>    for quick CLI profiling</>")
			color.Println("<fg=green>- </><fg=green;op=bold;>SPX_ENABLED=1 SPX_FP_LIVE=1</><fg=green> php ...</>")
			color.Println("<fg=green>    for quick CLI profiling with live redraw</>")
			color.Println("<fg=green>- </><fg=green;op=bold;>SPX_ENABLED=1 SPX_REPORT=full</><fg=green> php ...</>")
			color.Println("<fg=green>    for CLI profiling which can be analyzed in the web UI</>")
			color.Println("<fg=green>=====================================</>")
		},
	}

	phpProfilerCommand.Flags().SetInterspersed(false)
	phpProfilerCommand.Flags().StringVarP(&debugImage, "debug-image", "", "nicolaka/netshoot", "What debugger docker image to use for executing nsenter. By default, nicolaka/netshoot is used")

	return phpProfilerCommand
}
