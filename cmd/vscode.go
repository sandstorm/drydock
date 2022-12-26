package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gookit/color"
	"github.com/jonhadfield/findexec"
	"github.com/sandstorm/docker-execroot/util"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"syscall"
)

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

			dockerRunCommand := basicDockerRunCommand(fullContainerName, debugImage, pid)

			dockerRunCommand = append(dockerRunCommand, "/bin/bash", "-c", "mkdir /procContainer; mount -t proc proc /procContainer; ln -s /procContainer/1/root /container; chroot /container")

			obj := &VSCodeAttachedContainerT{
				ContainerName: "/" + fullContainerName + "_DEBUG",
			}
			bytes, err := json.Marshal(obj)
			encodedStr := hex.EncodeToString(bytes)

			containerToOpen := fmt.Sprintf("attached-container+%s %s", encodedStr, "/container"+containerPath)

			ensureImageExistsLocally(debugImage)

			c := exec.Command("/bin/bash", "-c", "sleep 1; code --remote "+containerToOpen)
			c.Start()

			color.Printf("<op=bold;>---------------------------------------------------------------------------</>\n")
			color.Printf("<op=bold;>Do not close this shell</> as long as you want to use VSCode in the container.\n")
			color.Printf("The connected container file system is mounted in <op=bold;>/container.</>\n")
			color.Printf("NOTE: the /proc file system of the connected container is mounted to\n")
			color.Printf("      /procContainer, because otherwise, VS Code does not work.\n")
			color.Printf("<op=bold;>---------------------------------------------------------------------------</>\n")
			syscall.Exec(dockerExecutablePathAndFilename, dockerRunCommand, os.Environ())
		},
	}

	return execRootCmd
}

func ensureImageExistsLocally(debugImage string) {
	img, err := util.ExecCommand("docker", "image", "ls", debugImage)
	if err == nil && len(img) > 0 {
		// we found the image locally
		return
	}

	pullCmd := exec.Command("docker", "pull", debugImage)
	pullCmd.Run()
}
