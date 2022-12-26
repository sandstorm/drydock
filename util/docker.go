package util

import (
	"bytes"
	"os/exec"
	"strings"
)

func TryGetDockerContainerNameFromDockerCompose(possibleComposeServiceName string) (string, error) {
	return ExecCommand("docker-compose", "ps", "-q", possibleComposeServiceName)
}

func ExecCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func dockerInspect(containerName string, format string) (string, error) {
	return ExecCommand("docker", "container", "inspect", "--format", format, containerName)
}

func GetRootPidForDockerContainer(containerName string) (string, error) {
	return dockerInspect(containerName, "{{.State.Pid}}")
}

func GetFullContainerName(containerName string) (string, error) {
	return dockerInspect(containerName, "{{.Name}}")
}
