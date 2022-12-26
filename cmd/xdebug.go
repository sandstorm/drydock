package cmd

import (
	"github.com/gookit/color"
	"github.com/jonhadfield/findexec"
	"github.com/sandstorm/docker-execroot/util"
	"github.com/spf13/cobra"
	"log"
	"net"
	"os"
	"os/exec"
	"time"
)

// phpSpxInstallScript is running in the debugImage (by default nicolaka/netshoot).
//   - we mount the inner container to /container (should be based on some base "official" Docker PHP image)
//   - we php-spx via Git inside nicolaka/netshoot (because we cannot know if git is installed inside the container)
//   - then, we compile and install php-spx inside the container. This runs as root, because we use the "execroot" mechanics
//     (important for the `make install` step).
//   - reload the config
const phpXdebugInstallScript = mountSlashContainer + `
cat << EOF | chroot /container
	pecl install xdebug
EOF

cat << EOF > /container$PHP_INI_DIR/conf.d/xdebug.ini
zend_extension=xdebug.so

xdebug.mode = develop,debug
xdebug.client_host = host.docker.internal
xdebug.discover_client_host = true
xdebug.max_nesting_level = 2048
EOF

pkill -USR2 php-fpm
`

func buildXdebugCommand() *cobra.Command {
	var debugImage string = "nicolaka/netshoot"

	var phpProfilerCommand = &cobra.Command{
		Use:   "xdebug [flags] SERVICE-or-CONTAINER",
		Short: "Install Xdebug in the given container",
		Long: color.Sprintf(`Usage:	docker xdebug [flags] SERVICE-OR-CONTAINER

Install Xdebug https://xdebug.org into the given PHP Container, and reloads
the PHP Process such that the debugger is enabled.

<op=underscore;>Options:</>
      --debug-image          What debugger docker image to use for executing nsenter and git.
                             By default, nicolaka/netshoot is used 

<op=underscore;>Examples</>

<op=bold;>Install Xdebug in a PHP container</>
	docker xdebug <op=italic;>myContainer</>

<op=bold;>Install Xdebug in a running docker-compose service</>
	docker xdebug <op=italic;>my-docker-compose-service</>

<op=underscore;>Background:</>

    This command installs the Xdebug PHP extension into an existing Docker container, even if the container is locked
    down to a non-root user. Additionally, we reload the PHP process by using kill -USR2.

    This command is using <op=italic;>nsenter</> wrapped in a privileged docker container to install the PHP extension
    inside a running container as root.

`),
		Args: cobra.MinimumNArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			//isOpen := isXdebugPortOpenInIde("127.0.0.1", "9003")

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
			color.Println("<green>Installing Xdebug into the container</>")
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
			dockerRunCommand = append(dockerRunCommand, phpXdebugInstallScript)

			c := exec.Command(dockerExecutablePathAndFilename, dockerRunCommand[1:]...)
			c.Env = os.Environ()
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin
			c.Run()

			color.Println("")
			color.Println("")
			color.Println("<fg=green>=====================================</>")
			color.Printf("<fg=green;op=bold>Finished installing xdebug into %s</>\n", fullContainerName)
			color.Println("")
			color.Println("<fg=green>Preparation: Set up PHPStorm/IntelliJ</>")
			color.Println("<fg=green>- </><fg=green;op=bold;>Run -> Start Listening for PHP Debug Connections</>")
			color.Println("<fg=green>    needs to be enabled; otherwise connection to the IDE does not work.</>")
			color.Println("")
			color.Println("")
			color.Println("<fg=green>Debugging Web Requests</>")
			color.Println("")
			color.Println("<fg=green>- </><fg=green;op=bold;>xdebug_break()</>")
			color.Println("<fg=green>    in PHP code to set a breakpoint (recommended for Neos/Flow)</>")
			color.Println("<fg=green>- http://your-url-here/</><fg=green;op=bold;>?XDEBUG_SESSION=1</>")
			color.Println("<fg=green>    in your HTTP request to debug a single request</>")
			color.Println("<fg=green>- http://your-url-here/</><fg=green;op=bold;>?XDEBUG_SESSION_START=1</>")
			color.Println("<fg=green>    in your HTTP request to debug all requests in this session</>")
			color.Println("<fg=green>    (stop with XDEBUG_SESSION_STOP=1)</>")
			color.Println("")
			color.Println("<fg=green>Debugging CLI requests:</>")
			color.Println("")
			color.Println("<fg=green>- </><fg=green;op=bold;>XDEBUG_SESSION=1</><fg=green> php ...</>")
			color.Println("<fg=green>    for CLI Step Debugging</>")
			color.Println("")
			color.Println("")
			color.Println("<fg=green>Setting Up Path Mappings in PHPStorm/IntelliJ</>")
			color.Println("")
			color.Println("<fg=green>You need to set up path mappings correctly, otherwise you cannot navigate</>")
			color.Println("<fg=green>to the files in the IDE when a breakpoint is hit. This can be done as follows:</>")
			color.Println("")
			color.Println("<fg=green>- When a breakpoint is hit:</>")
			color.Println("<fg=green>    Debug Panel -> Threads&Variables</>")
			color.Println("<fg=green>    -> </><fg=green;op=bold;>Click to set up path mappings</>")
			color.Println("<fg=green>- When the path mapping is wrongly configured and you need to correct it:</>")
			color.Println("<fg=green>    Settings</>")
			color.Println("<fg=green>    -> Languages&Frameworks -> PHP -> Server</>")
			color.Println("<fg=green>    -> (add server if needed)</>")
			color.Println("<fg=green>    -> </><fg=green;op=bold;>Use Path Mappings</>")
			color.Println("")
			color.Println("<fg=green>There might still be problems debugging Neos/Flow...</>")
			color.Println("<fg=green>=====================================</>")
		},
	}

	phpProfilerCommand.Flags().SetInterspersed(false)
	phpProfilerCommand.Flags().StringVarP(&debugImage, "debug-image", "", "nicolaka/netshoot", "What debugger docker image to use for executing nsenter. By default, nicolaka/netshoot is used")

	return phpProfilerCommand
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
