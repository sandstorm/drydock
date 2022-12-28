package cmd

const mountSlashContainer = "mount -t proc proc /proc; ln -s /proc/1/root /container;"

func dockerRunNsenterCommand(fullContainerName, debugImage, pid string, extraDockerRunArgs []string) []string {
	result := dockerRunCommand(fullContainerName, debugImage, extraDockerRunArgs)
	result = append(result,
		nsenterCommand(pid)...,
	)
	return result
}

func dockerRunCommand(fullContainerName, debugImage string, extraDockerRunArgs []string) []string {
	result := []string{
		"docker", "run",
		"--rm", // ephemeral container
		"-it",  // interactive, with TTY
		"--name",
		fullContainerName + "_DEBUG",
		"--privileged", // we need privileged permissions to run nsenter (namespace enter), to enter the other container
		"--pid=host",   // we need to see the *hosts* PIDs, so that nsenter can enter the correct container
	}

	result = append(result, extraDockerRunArgs...)

	result = append(result,
		debugImage,
	)
	return result
}

func nsenterCommand(pid string) []string {
	return []string{
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
