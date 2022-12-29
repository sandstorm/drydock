# `drydock spx` - install PHP-SPX profiler in a running PHP Docker container

## Background

We've evaluated quite some PHP Profilers and finally settled on [PHP-SPX](https://github.com/NoiseByNorthwest/php-spx),
which is a PHP extension with a great embedded profiling UI. Most other open source profilers only have
a recorder, but not a good user interface for browsing the recorded values.

We often use the same Docker image for development and production, to have dev-prod parity and reduce complexity.
Our docker images are tuned for readability; so they are as short as possible and we try not to go overboard
with advanced features like layered builds. Our Dockerfiles are included in each project repository, as this gives
us good stability and customizability (we usually need that).

We don't want tools such as SPX installed in the production image, and it would be quite a pain to maintain such a set
of tools for *every* Dockerfile in every project.

**`drydock spx` installs SPX into a running PHP container.**

## Prerequisites

PHP extensions must be compilable inside your container; so `phpize; ./configure; make` must work.
All images which are based on [the official PHP base image](https://hub.docker.com/_/php) satisfy this requirement.

## Usage

```bash
drydock spx [container-name]
drydock spx [docker-compose-name]
```

Convenience: You can either specify a container name, or also a `docker-compose` service name if you run this in a
folder with a `docker-compose.yml` file inside).

After a few seconds, usage instructions are printed, containing all relevant URLs:

```
=====================================
Finished installing PHP-SPX into /neos-on-docker-kickstart-neos-1

SPX Profiler URL:
  - http://127.0.0.1:8081/?SPX_UI_URI=/&SPX_KEY=dev
  - http://127.0.0.1:9090/?SPX_UI_URI=/&SPX_KEY=dev

Profiling CLI requests:
- SPX_ENABLED=1 php ...
    for quick CLI profiling
- SPX_ENABLED=1 SPX_FP_LIVE=1 php ...
    for quick CLI profiling with live redraw
- SPX_ENABLED=1 SPX_REPORT=full php ...
    for CLI profiling which can be analyzed in the web UI
=====================================
```

## Help Text

```
drydock spx SERVICE-OR-CONTAINER

Install the SPX PHP-Profiler https://github.com/NoiseByNorthwest/php-spx into the given PHP Container, and reloads
the PHP Process such that the profiler is enabled.

Options:
      --debug-image          What debugger docker image to use for executing nsenter.
                             By default, nicolaka/netshoot is used

Examples

Install PHP-SPX Profiler in a PHP container
	drydock spx myContainer

Install PHP-SPX Profiler in a running docker-compose service
	drydock spx my-docker-compose-service

Background:

    This command installs the php-spx PHP extension into an existing Docker container, even if the container is locked
    down to a non-root user. Additionally, we reload the PHP process by using kill -USR2.

    This command is using nsenter wrapped in a privileged docker container to install the PHP extension
    inside a running container as root.
```
