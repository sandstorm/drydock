package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
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

func GetEnvCliCallsForDockerRunFromContainerMetadata(containerName string) ([]string, error) {
	var dockerRunCommand []string
	envAsJsonString, err := dockerInspect(containerName, "{{json .Config.Env}}")
	if err != nil {
		return nil, fmt.Errorf("could not run docker inspect: %w", err)
	}

	var parsedEnv []string
	err = json.Unmarshal([]byte(envAsJsonString), &parsedEnv)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON result of docker inspect %s - nested error: %w", envAsJsonString, err)
	}

	for _, s := range parsedEnv {
		dockerRunCommand = append(dockerRunCommand, "--env", s)
	}

	return dockerRunCommand, nil
}

type dockerPorts struct {
	HostIp   string
	HostPort string
}

func GetHostPorts(containerName string) ([]int, error) {
	envAsJsonString, err := dockerInspect(containerName, "{{json .NetworkSettings.Ports}}")
	if err != nil {
		return nil, fmt.Errorf("could not run docker inspect: %w", err)
	}

	var parsedEnv map[string][]dockerPorts
	err = json.Unmarshal([]byte(envAsJsonString), &parsedEnv)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON result of docker inspect %s - nested error: %w", envAsJsonString, err)
	}

	var result []int
	for _, s := range parsedEnv {
		for _, inner := range s {
			tmp, _ := strconv.Atoi(inner.HostPort)
			result = append(result, tmp)
		}
	}

	return result, nil
}

func GetFullContainerName(containerName string) (string, error) {
	return dockerInspect(containerName, "{{.Name}}")
}
