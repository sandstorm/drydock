# docker execroot, docker vscode, docker phpprofiler extensions for Docker CLI

## Installation

1. Run the following command to install via homebrew:

   ```bash
   brew install sandstorm/tap/docker-execroot
   ```

2. Add the correct symlinks

   ```bash
   mkdir -p ~/.docker/cli-plugins
   rm -f ~/.docker/cli-plugins/docker-execroot
   ln -s /usr/local/opt/docker-execroot/share/docker-execroot/docker-execroot ~/.docker/cli-plugins/docker-execroot
   
   rm -f ~/.docker/cli-plugins/docker-vscode
   ln -s /usr/local/opt/docker-execroot/share/docker-execroot/docker-execroot ~/.docker/cli-plugins/docker-vscode
   
   rm -f ~/.docker/cli-plugins/docker-phpprofiler
   ln -s /usr/local/opt/docker-execroot/share/docker-execroot/docker-execroot ~/.docker/cli-plugins/docker-phpprofiler

   rm -f ~/.docker/cli-plugins/docker-xdebug
   ln -s /usr/local/opt/docker-execroot/share/docker-execroot/docker-execroot ~/.docker/cli-plugins/docker-xdebug

   ```

3. Profit!

   ```
   docker execroot -h
   docker vscode -h
   docker phpprofiler -h
   ```

### `docker execroot myContainer`

- Run `docker execroot` is like `docker exec` as root; no matter what user is configured in the `Dockerfile` or in `docker-compose.yml`.
- This is helpful for debugging, when you want to install additional tools in the container, or edit files not normally
  editable to your run-user.
- Convenience: You can either specify a container name, or also a `docker-compose` service name.
- With `--no-chroot`, you can bring your own *Debug* container with your custom tools; e.g. useful to debug containers
  which start from `scratch` as base image (like Golang tools)

### `docker vscode`

- Prerequisites:
  - Install [Visual Studio Code](https://code.visualstudio.com/) 
  - Install the [Visual Studio Code Remote - Containers](https://aka.ms/vscode-remote/download/containers) extension
  - Install the command-line `code` launcher [as explained here](https://code.visualstudio.com/docs/setup/mac#_launching-from-the-command-line)
- You can run `docker vscode [containername]` to launch a connected VS Code instance.

### `docker phpprofiler`

This installs the [PHP-SPX](https://github.com/NoiseByNorthwest/php-spx) profiler into a *running* PHP Application
in Docker; and outputs usage instructions at the end to get you up and running quickly.

- This works with containers extending from the default Docker Hub [php](https://hub.docker.com/_/php) images,
  which are also set up to compile PHP extensions.
- Convenience: You can either specify a container name, or also a `docker-compose` service name.
- Usage: `docker phpprofiler [containername]` to add PHP-SPX to the running container. 


## Full Usage Instructions

### `docker execroot [flags] SERVICE-OR-CONTAINER COMMAND [ARG...]`

```
Run a command AS ROOT in a running container or docker-compose service.

Options:
      --no-chroot            Do not enter the target container file system, but stay in the
                             debug-image. Target container is mounted in /container
      --debug-image          What debugger docker image to use for executing nsenter.
                             By default, nicolaka/netshoot is used

Examples

Get a root shell in a running container
	docker execroot myContainer

Get a root shell in a running docker-compose service
	docker execroot my-docker-compose-service

Execute a command as root
	docker execroot myContainer whoami

Stay in the debug container instead of entering the target container
	docker execroot --no-chroot myContainer

Change the debug container
	docker execroot --no-chroot --debug-image=alpine myContainer

Using VSCode to:

Background:

    docker-compose exec or docker exec respect the USER specified in the Dockerfile; and it is
    not easily possible to break out of this user (e.g. to install an additional tool as root).

    This command is using nsenter wrapped in a privileged docker container to enter a running container as root.
```

### `docker vscode SERVICE-OR-CONTAINER [PATH]`

```
Open VSCode Remote Containers as root; at path [PATH].

Examples

Open VSCode as root user
	docker vscode myContainer

Open a specific folder in VSCode as root user
	docker vscode myContainer /app

Usage:
  docker vscode SERVICE-or-CONTAINER [PATH] [flags]

Flags:
  -h, --help   help for vscode
```

### `docker phpprofiler SERVICE-OR-CONTAINER`

```
Install the SPX PHP-Profiler https://github.com/NoiseByNorthwest/php-spx into the given PHP Container, and reloads
the PHP Process such that the profiler is enabled.

Options:
      --debug-image          What debugger docker image to use for executing nsenter.
                             By default, nicolaka/netshoot is used

Examples

Install PHP-SPX Profiler in a PHP container
	docker phpprofiler myContainer

Install PHP-SPX Profiler in a running docker-compose service
	docker phpprofiler my-docker-compose-service

Background:

    This command installs the php-spx PHP extension into an existing Docker container, even if the container is locked
    down to a non-root user. Additionally, we reload the PHP process by using kill -USR2.

    This command is using nsenter wrapped in a privileged docker container to install the PHP extension
    inside a running container as root.
```

## Developing

Simply have a modern Go version installed; check out the project somewhere (NOT in $GOPATH, as we use Go Modules),
and then run `./build.sh`.

## Releasing new versions


### Prerequisites for releasing

1. ensure you have [goreleaser](https://goreleaser.com/) installed:

  ```bash
  brew install goreleaser/tap/goreleaser
  ```

2. Create a new token for goreleaser [in your GitHub settings](https://github.com/settings/tokens); select the `repo` scope.

3. put the just-created token into the file `~/.config/goreleaser/github_token`



### Doing the release

Testing a release:

```
goreleaser --snapshot --skip-publish --rm-dist --debug
```

Executing a release:

1. Commit all changes, create a new tag and push it.

```
TAG=v0.9.0 git tag $TAG; git push origin $TAG
```

2. run goreleaser:

```
goreleaser --rm-dist
```

## Inspiration and resources

- [Docker CLI Plugins 1](https://gist.github.com/thaJeztah/b7950186212a49e91a806689e66b317d)
- [Docker CLI Plugins 2](https://dille.name/slides/2019-06-06/020_advanced/080_docker_cli_plugins/slides/)
- https://enqueuezero.com/container-and-nsenter.html
- https://collabnix.com/how-to-access-docker-namespace/
- https://dev.to/stefanjacobs/nsenter-entering-a-running-process-or-container-46a3
- https://man7.org/linux/man-pages/man1/nsenter.1.html
- https://github.com/nicolaka/netshoot

## License

MIT
