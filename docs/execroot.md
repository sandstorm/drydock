# `drydock execroot myContainer` - like `sudo` for your containers

## Background

It is a Docker best practice to run as an unprivileged user inside the Container. This makes
development more complicated, as you cannot easily install additional tools into the container
without rebuilding it. For me personally, a quick development iteration time is crucial; so before
adjusting a `Dockerfile`, I often try out config changes or package installs inside a running container
without rebuilding it. This does not work as unprivileged user; and I often wanted some kind of `sudo` mode
inside my containers.

**`drydock execroot` is like `sudo` for containers: It spawns a root shell into a running container.**

## Usage

Spawn a root shell into a running container:
```bash
drydock execroot [container-name]
drydock execroot [docker-compose-name]
drydock execroot [container-name] apt-get install vim
```

Convenience: You can either specify a container name, or also a `docker-compose` service name if you run this in a
folder with a `docker-compose.yml` file inside).

## Advanced Usage

`drydock` works by creating a debugging sidecar container with elevated permissions, and then switching to the
target container. If you don't want this but stay in the sidecar container, use the `--no-chroot` option.

The target container is mounted as `/container`. As the debug container shares the network and process namespaces
of the target container, you can reach your HTTP services via `127.0.0.1` and you can view all processes of the
target container via `ps -ef`.

By default, we use `nicolaka/netshoot` as debug container, because that has lots of valuable tools installed.
You can specify another debug container with `--debug-image`. This image needs `nsenter` installed.

Using `--no-chroot` and optionally another debug container is especially useful when to debug containers
which start from `scratch` as base image (like Golang tools).

## Help Text

```
drydock execroot [flags] SERVICE-OR-CONTAINER COMMAND [ARG...]


Run a command AS ROOT in a running container or docker-compose service.

Options:
      --no-chroot            Do not enter the target container file system, but stay in the
                             debug-image. Target container is mounted in /container
      --debug-image          What debugger docker image to use for executing nsenter.
                             By default, nicolaka/netshoot is used

Examples

Get a root shell in a running container
	drydock execroot myContainer

Get a root shell in a running docker-compose service
	drydock execroot my-docker-compose-service

Execute a command as root
	drydock execroot myContainer whoami

Stay in the debug container instead of entering the target container
	drydock execroot --no-chroot myContainer

Change the debug container
	drydock execroot --no-chroot --debug-image=alpine myContainer

Using VSCode to:

Background:

    docker-compose exec or docker exec respect the USER specified in the Dockerfile; and it is
    not easily possible to break out of this user (e.g. to install an additional tool as root).

    This command is using nsenter wrapped in a privileged docker container to enter a running container as root.
```
