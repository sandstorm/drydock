/*
Copyright © 2020 Sebastian Kurfuerst

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gookit/color"
	"github.com/jonhadfield/findexec"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var cfgFile string

func execCommand(command string, args... string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func tryGetDockerContainerNameFromDockerCompose(possibleComposeServiceName string) (string, error) {
	return execCommand("docker-compose", "ps", "-q", possibleComposeServiceName)
}

func dockerInspect(containerName string, format string) (string, error) {
	return execCommand("docker", "container", "inspect", "--format", format, containerName)
}

func getRootPidForDockerContainer(containerName string) (string, error) {
	return dockerInspect(containerName, "{{.State.Pid}}")
}

func getFullContainerName(containerName string) (string, error) {
	return dockerInspect(containerName, "{{.Name}}")
}

var rootCmd = &cobra.Command{
	Use: "docker",
}

func basicDockerRunCommand(fullContainerName, debugImage, pid string) []string {
	return []string{
		"docker", "run",
		"--rm", // ephemeral container
		"-it",  // interactive, with TTY
		"--name",
		fullContainerName + "_DEBUG",
		"--privileged", // we need privileged permissions to run nsenter (namespace enter), to enter the other container
		"--pid=host",   // we need to see the *hosts* PIDs, so that nsenter can enter the correct container
		debugImage,
		// here, the "nsenter" invocation follows
		"nsenter",
		"--target", pid, // we want to attach to the found target PID
		// we want to share the network namespace. This means you can e.g. use `curl` like in the debugged application,
		// using "127.0.0.1:[yourport]" as usual.
		"--net",
		// IPC seems necessary, but not 100% sure why.
		"--ipc",
		// we want to share the PID namespace. This means:
		// - by default, "ps -ef" in the container is showing the host namespace, because "ps -ef" is looking at the /proc file system.
		// - by running "mount -t proc proc /proc", we get the proc file system of the TARGET namespace (i.e. the container we want to debug).
		//   -> at this point, "ps -ef" displays the OTHER processes.
		"--pid",
	}
}

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

<op=underscore;>Using VSCode to:</>

<op=underscore;>Background:</>

    <op=italic;>docker-compose exec</> or <op=italic;>docker exec</> respect the USER specified in the Dockerfile; and it is
    not easily possible to break out of this user (e.g. to install an additional tool as root).

    This command is using <op=italic;>nsenter</> wrapped in a privileged docker container to enter a running container as root.
`),
		Args: cobra.MinimumNArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			dockerContainerIdentifier, err := tryGetDockerContainerNameFromDockerCompose(args[0])

			if err != nil {
				// could not identify the docker container identifier; e.g. no docker-compose used.
				dockerContainerIdentifier = args[0]
			} else {
				log.Printf("docker compose service '%s' found, entering it.\n", args[0])
			}

			pid, err := getRootPidForDockerContainer(dockerContainerIdentifier)

			if err != nil {
				// container not running
				log.Printf("FATAL: Container '%s' not running\n", dockerContainerIdentifier)
				os.Exit(1)
			}

			fullContainerName, err := getFullContainerName(dockerContainerIdentifier)
			if err != nil {
				log.Printf("FATAL: Could not extract container name for container '%s' - THIS SHOULD NOT HAPPEN. Please file a bug report.\n", dockerContainerIdentifier)
				os.Exit(1)
			}

			dockerExecutablePathAndFilename := findexec.Find("docker", "")

			dockerRunCommand := basicDockerRunCommand(fullContainerName, debugImage, pid)

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
					dockerRunCommand = append(dockerRunCommand, "/bin/bash", "-c", "mount -t proc proc /proc; ln -s /proc/1/root /container; /bin/bash -l")

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

type VSCodeAttachedContainerT struct {
	ContainerName string `json:"containerName"`
}

func buildVsCodeCommand() *cobra.Command {
	var debugImage string = "nicolaka/netshoot"

	var execRootCmd = &cobra.Command{
		Use:   "vscode SERVICE-or-CONTAINER [PATH]",
		Short: "Opens VScode to a container, where you can edit all files (because using root)",
		Long: color.Sprintf(`Usage:	docker vscode SERVICE-OR-CONTAINER [PATH]

Open VSCode Remote Containers as root; at path [PATH].

<op=underscore;>Examples</>

<op=bold;>Open VSCode as root user</>
	docker vscode <op=italic;>myContainer</>

<op=bold;>Open a specific folder in VSCode as root user</>
	docker vscode <op=italic;>myContainer</> /app
`),
		Args: cobra.RangeArgs(1, 2),

		Run: func(cmd *cobra.Command, args []string) {
			containerPath := "/"
			if len(args) == 2 {
				containerPath = args[1]
			}
			dockerContainerIdentifier, err := tryGetDockerContainerNameFromDockerCompose(args[0])

			if err != nil {
				// could not identify the docker container identifier; e.g. no docker-compose used.
				dockerContainerIdentifier = args[0]
			} else {
				log.Printf("docker compose service '%s' found, entering it.\n", args[0])
			}

			pid, err := getRootPidForDockerContainer(dockerContainerIdentifier)

			if err != nil {
				// container not running
				log.Printf("FATAL: Container '%s' not running\n", dockerContainerIdentifier)
				os.Exit(1)
			}

			fullContainerName, err := getFullContainerName(dockerContainerIdentifier)
			if err != nil {
				log.Printf("FATAL: Could not extract container name for container '%s' - THIS SHOULD NOT HAPPEN. Please file a bug report.\n", dockerContainerIdentifier)
				os.Exit(1)
			}

			dockerExecutablePathAndFilename := findexec.Find("docker", "")

			dockerRunCommand := basicDockerRunCommand(fullContainerName, debugImage, pid)

			dockerRunCommand = append(dockerRunCommand, "/bin/bash", "-c", "mkdir /procContainer; mount -t proc proc /procContainer; ln -s /procContainer/1/root /container; chroot /container")

			obj := &VSCodeAttachedContainerT{
				ContainerName: "/" + fullContainerName + "_DEBUG",
			}
			bytes, err := json.Marshal(obj)
			encodedStr := hex.EncodeToString(bytes)

			containerToOpen := fmt.Sprintf("attached-container+%s %s", encodedStr, "/container" + containerPath)

			ensureImageExistsLocally(debugImage)


			c := exec.Command("/bin/bash", "-c", "sleep 1; code --remote " + containerToOpen)
			c.Start()

			syscall.Exec(dockerExecutablePathAndFilename, dockerRunCommand, os.Environ())
		},
	}

	return execRootCmd
}

func ensureImageExistsLocally(debugImage string) {
	img, err := execCommand("docker", "image", "ls", debugImage)
	if err == nil && len(img) > 0 {
		// we found the image locally
		return
	}

	pullCmd := exec.Command("docker", "pull", debugImage)
	pullCmd.Run()
}

// see https://pkg.go.dev/github.com/docker/cli/cli-plugins/manager#Metadata
type DockerPluginMetadata struct {
	// SchemaVersion describes the version of this struct. Mandatory, must be "0.1.0"
	SchemaVersion string `json:",omitempty"`
	// Vendor is the name of the plugin vendor. Mandatory
	Vendor string `json:",omitempty"`
	// Version is the optional version of this plugin.
	Version string `json:",omitempty"`
	// ShortDescription should be suitable for a single line help message.
	ShortDescription string `json:",omitempty"`
	// URL is a pointer to the plugin's homepage.
	URL string `json:",omitempty"`
}

func buildDockerCliPluginMetadata(version, commit string) *cobra.Command {
	return &cobra.Command{
		Use:    "docker-cli-plugin-metadata",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {

			desc := "Run a command in a running container as ROOT user"
			println(os.Args[0])
			if len(os.Args) >= 1 && strings.Contains(os.Args[0], "docker-vscode") {
				desc = "Open VSCode as container"
			}

			metadata := DockerPluginMetadata{
				SchemaVersion:    "0.1.0",
				Vendor:           "sandstorm",
				Version:          fmt.Sprintf("%s - %s", version, commit),
				ShortDescription: desc,
			}
			res, err := json.Marshal(metadata)
			if err != nil {
				log.Fatalf("Error building up plugin metadata - should never happen: %s", err)
			}
			fmt.Println(string(res))
		},
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version, commit string) {
	rootCmd.AddCommand(buildDockerCliPluginMetadata(version, commit))
	rootCmd.AddCommand(buildExecRootCmd())
	rootCmd.AddCommand(buildVsCodeCommand())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()
}
