# `drydock vscode myContainer` - edit any file with Visual Studio Code

## Background

It is a Docker best practice to run as an unprivileged user inside the Container. Sometimes, I had
trouble using Visual Studio Code Remote Containers in these scenarios.

**`drydock vscode` opens your container as root in Visual Studio Code, allowing to edit arbitrary files.**

## Prerequisites

- Install [Visual Studio Code](https://code.visualstudio.com/)
- Install the [Visual Studio Code Remote - Containers](https://aka.ms/vscode-remote/download/containers) extension
- Install the command-line `code` launcher [as explained here](https://code.visualstudio.com/docs/setup/mac#_launching-from-the-command-line)

## Usage

```bash
drydock vscode [container-name]
drydock vscode [docker-compose-name]
drydock vscode [container-name] /usr/lib
```

Convenience: You can either specify a container name, or also a `docker-compose` service name if you run this in a
folder with a `docker-compose.yml` file inside).

If you specify a path inside your container as second argument, this is the folder which is opened in VS Code.

## Help Text

```
drydock vscode SERVICE-OR-CONTAINER [PATH]


Open VSCode Remote Containers as root; at path [PATH].

Examples

Open VSCode as root user
	drydock vscode myContainer

Open a specific folder in VSCode as root user
	drydock vscode myContainer /app

Usage:
  drydock vscode SERVICE-or-CONTAINER [PATH] [flags]

Flags:
  -h, --help   help for vscode
```
