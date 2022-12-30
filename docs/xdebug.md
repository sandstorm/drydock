# `drydock xdebug myContainer` - run Xdebug step debugger in a running PHP container

## Background

[Xdebug](https://xdebug.org) is the standard PHP debugger (amongst other things) with great PHPStorm/IntelliJ
IDE integration. Right now (as explained in [drydock spx](spx.md)), we cannot include Xdebug in our Docker images,
because we want to use the same images for dev and prod; and xdebug comes with a performance penalty.

**`drydock xdebug` temporarily runs Xdebug in a running PHP container.**

## Prerequisites

- In IntelliJ/PHPStorm you need to enable `Run -> Start Listening for PHP Debug Connections`.
- For Neos/Flow: In case you use the `--mount=app/Data/Temporary,app/Packages` option (read for details below),
  it is beneficial to mark the `Data` folder as excluded, and enable `File -> Power Save Mode` to prevent
  long-running re-indexing in the IDE.

## Usage

```bash
drydock xdebug [container-name]
drydock xdebug [docker-compose-name]
# for Neos/Flow applications, additionally mount Data/Temporary, and the full Packages folder to enable
# editing *any* file. NOTE: To prevent PHPStorm from freezing, enable `File -> Power Save Mode` while debugging.
drydock xdebug [container-name] --mount=app/Data/Temporary,app/Packages
```

Convenience: You can either specify a container name, or also a `docker-compose` service name if you run this in a
folder with a `docker-compose.yml` file inside).

## Advanced Usage: extra mounts, f.e. for Neos/Flow

**tl;dr: If you use Neos/Flow with the `DistributionPackages` mounted into the Docker container,
use `drydock xdebug [container-name] --mount=app/Data/Temporary,app/Packages`.**

**If you use Neos/Flow with the `DistributionPackages` *and* `Packages` mounted into the Docker container,
use `drydock xdebug [container-name] --mount=app/Data/Temporary`.**

Read on for the full explanation.

Because mounting lots of files from Mac OS to Docker comes with a certain performance penalty, we often do not
mount the full Neos/Flow project into the container. With newest Docker and *VirtioFS*, the situation has improved
tremendously.

Before VirtioFS, we have used a directory layout like the following:

```
**Old Neos Mount Structure**

/app
    composer.json
    DistributionPackages/ <-- MOUNTED from the host (as the only folder)
      My.Package
    Packages/
      Framework/
      Application/
        My.Package <-- Symlink to DistributionPackages/My.Package
    Data/
      Temporary/
```

With modern Docker, we often also mount the full `Packages` folder (which still comes with a performance cost,
but this is OKish):

```
**New Neos Mount Structure**

/app
    composer.json
    DistributionPackages/ <-- MOUNTED from the host
      My.Package
    Packages/ <-- MOUNTED from the host (additionally)
      Framework/
      Application/
        My.Package <-- Symlink to DistributionPackages/My.Package
    Data/
      Temporary/
```

Neos/Flow compiles most PHP files into a Code Cache inside `Data/Temporary`. Even with most modern Docker
and VirtioFS, it is too slow to mount the full `Data/Temporary` folder.

For debugging to work, the IDE needs to open the temporary files because if you put a breakpoint
in with `xdebug_break()`, these breakpoints appear in the Data/Temporary files.

Thus, we've come up with an extra `--mount` option which allow to mount folders **from the container to the host**
(so that is the opposite direction as usual) - and this way, we can make the cached classes available
to PHPStorm/IntelliJ. Then, Xdebug debugging will properly work.

Internally we're starting a webdav server in the sidecar debug container, and mount the share via Webdav on OSX.

## Help Text

```
drydock xdebug [flags] SERVICE-OR-CONTAINER

Run Xdebug https://xdebug.org in the given PHP Container, and reloads
the PHP Process such that the debugger is enabled.

Options:
      --debug-image          What debugger docker image to use for executing nsenter (and optionally the NFS webdav server).
                             By default, nicolaka/netshoot is used
      --mount                Extra mounts which should be mounted from the container to the host via webdav.
                             This is useful to be able to f.e. debug into non-mounted files (like other packages
                             in Neos/Flow applications)

Examples

Run Xdebug in a running PHP container
	drydock xdebug myContainer

Run Xdebug in a running docker-compose service
	drydock xdebug my-docker-compose-service

Run Xdebug a Neos/Flow Application
	drydock xdebug my-docker-compose-service --mount=app/Data/Temporary,Packages

Background:

    This command installs the Xdebug PHP extension into an existing Docker container, even if the container is locked
    down to a non-root user. Additionally, we reload the PHP process by using kill -USR2.

    This command is using nsenter wrapped in a privileged docker container to install the PHP extension
    inside a running container as root.
```
