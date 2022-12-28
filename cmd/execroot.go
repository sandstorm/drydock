package cmd

import (
	"github.com/gookit/color"
	"github.com/jonhadfield/findexec"
	"github.com/sandstorm/docker-execroot/util"
	"github.com/spf13/cobra"
	"log"
	"os"
	"syscall"
)

func buildExecRootCmd() *cobra.Command {
	var noMount bool = false // by default, we mount the target
	var debugImage string = "nicolaka/netshoot"

	var execRootCmd = &cobra.Command{
		Use:   "execroot [flags] SERVICE-or-CONTAINER COMMAND [ARG...]",
		Short: "executes a command or an interactive shell ('docker-compose exec' or 'docker exec'), but enters the container as root in all cases",
		Long: color.Sprintf(`Usage:	docker execroot [flags] SERVICE-OR-CONTAINER COMMAND [ARG...]

Run a command AS ROOT in a running container or docker-compose service.

<op=underscore;>Options:</>
      --no-chroot            Do not enter the target container file system, but stay in the
                             debug-image. Target container is mounted in /container 
      --debug-image          What debugger docker image to use for executing nsenter.
                             By default, nicolaka/netshoot is used 

<op=underscore;>Examples</>

<op=bold;>Get a root shell in a running container</>
	docker execroot <op=italic;>myContainer</>

<op=bold;>Get a root shell in a running docker-compose service</>
	docker execroot <op=italic;>my-docker-compose-service</>

<op=bold;>Execute a command as root</>
	docker execroot <op=italic;>myContainer</> whoami

<op=bold;>Stay in the debug container instead of entering the target container</>
	docker execroot --no-chroot <op=italic;>myContainer</>

<op=bold;>Change the debug container</>
	docker execroot --no-chroot --debug-image=alpine <op=italic;>myContainer</>

<op=underscore;>Background:</>

    <op=italic;>docker-compose exec</> or <op=italic;>docker exec</> respect the USER specified in the Dockerfile; and it is
    not easily possible to break out of this user (e.g. to install an additional tool as root).

    This command is using <op=italic;>nsenter</> wrapped in a privileged docker container to enter a running container as root.
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

			// we need to get the ENV of the original container, needed such that f.e. "docker-php-ext-enable" will work: https://github.com/docker-library/php/blob/67c242cb1529c70a3969a373ab333c53001c95b8/8.2-rc/bullseye/cli/docker-php-ext-enable
			envVars, err := util.GetEnvCliCallsForDockerRunFromContainerMetadata(fullContainerName)
			if err != nil {
				log.Printf("FATAL: Could not extract env variables for container '%s': %s - THIS SHOULD NOT HAPPEN. Please file a bug report.\n", dockerContainerIdentifier, err)
				os.Exit(1)
			}

			dockerRunCommand := dockerRunNsenterCommand(fullContainerName, debugImage, pid, envVars)

			// OPTIONAL: "mount" does not need to be specified here.
			// - if leaving it OUT, the file system (and all tooling) is still from the nicolaka/netshoot container.
			//   - you can thereby use e.g. "vim" or other tools NOT installed in the container itself for debugging.
			//   - by then running "mount -t proc proc /proc", you can go to /proc/1/root and this is the IN-DEBUGGED-CONTAINER file system
			// - by INCLUDING it, "/proc" is automatically mounted (as well as all other FS), so that means you get all tooling
			//   from within the debugged container. As you are root, you can install new packages.
			if noMount {
				// do not mount, so "advanced" debug mode
				if len(args) > 1 {
					// command included in the args; so let's add it here.
					dockerRunCommand = append(dockerRunCommand, args[1:]...)
					color.Printf("<op=bold;>-----------------------------------------------------------------------------------</>\n")
					color.Printf("You can run <op=bold;>mount -t proc proc /proc</> to mount the proc filesystem of the container.\n")
					color.Printf("Afterwards, <op=bold;>/proc/1/root</> contains the debugged container file system.\n")
					color.Printf("<op=bold;>-----------------------------------------------------------------------------------</>\n")
				} else {
					// no command; so let's do a default where we mount the proc filesystem; and mount the target file system into /container
					dockerRunCommand = append(dockerRunCommand, "/bin/bash", "-c", mountSlashContainer+"/bin/bash -l")

					color.Printf("<op=bold;>-----------------------------------------------------------</>\n")
					color.Printf("The debugged container file system is mounted in <op=bold;>/container</>\n")
					color.Printf("<op=bold;>-----------------------------------------------------------</>\n")
				}
			} else {
				// mount by default; illusion of "root container"
				dockerRunCommand = append(dockerRunCommand, "--mount")

				if len(args) > 1 {
					// command included in the args; so let's add it here.
					dockerRunCommand = append(dockerRunCommand, args[1:]...)
				} else {
					dockerRunCommand = append(dockerRunCommand, "/bin/bash")
				}
			}

			syscall.Exec(dockerExecutablePathAndFilename, dockerRunCommand, os.Environ())
		},
	}

	execRootCmd.Flags().SetInterspersed(false)
	execRootCmd.Flags().BoolVarP(&noMount, "no-chroot", "", false, "Do not enter the target container file system, but stay in the debug-image. Target container is mounted in /container")
	execRootCmd.Flags().StringVarP(&debugImage, "debug-image", "", "nicolaka/netshoot", "What debugger docker image to use for executing nsenter. By default, nicolaka/netshoot is used")

	return execRootCmd
}
