# Architecture

This document will give some background on implementation details.

## How does the root-shell magic work?

Getting a root shell involves a few steps, explained here.

### 1. nsenter

The core idea is built around `nsenter`, which is a linux tool to attach to namespaces of processes.

an `nsenter` invocation looks as follows - we'll explain the pieces below:

```
nsenter \
    --target 12345 \
    --ipc \
    --pid \
    --net
```

`12345` in the example above is the Process ID of the process we want to enter, **as seen from the host**.

Remember the init process has always PID `1`; but from the outside (on the host system), the process has a
different ID, as it can be seen in the following example:

```
 **PID on the host system**                                       **PID inside the container**   
 2754 \_ php-fpm: master process (/usr/local/etc/php-fpm.conf)     1 php-fpm: master process (/usr/local/etc/php-fpm.conf)  
 3515     \_ nginx: master process nginx                         253 nginx: master process nginx
 3516     |   \_ nginx: worker process                           254 nginx: worker process
 3517     |   \_ nginx: worker process                           255 nginx: worker process
25953     \_ php-fpm: pool www                                 18970 php-fpm: pool www
25954     \_ php-fpm: pool www                                 18971 php-fpm: pool www
```

If we want to join this container, we need to specify the PID as seen from the host, so `2754` in the example above.
This PID can be found by using `docker inspect`:

```json
{
  "Id": "0e2c4bf2a9f91cb26fdbeafa13a92b5b1ce0cee263f16c25a9c987613aea0fa5",
  "State": {
    "Pid": 2754, // <-- we look for this PID 
  }
}
```

So, if we are on the host machine, we can run the `nsenter` command above, with the following details:

- entering the IPC (Interprocess Communication) namespace seems necessary, I am not 100% sure why.
- We want to share the `pid` namespace, because this shows us the same process listing as within the container.
  (and we need to remember this trick a for a second, as it enables entering the root file system - see below.)
- We usually want to enter the `net` (network) namespace, so that we can reach the running services via
  `127.0.0.1`.

We could also join the `mount` namespace. This way, we would also get the root file system of the container.
We do *not* do this, because we want to be able to have a set of extra development / debugging tools from the
debug container available.

### Making the root filesystem of the container available

Because we have access to the `pid` namespace, we mount the `/proc` filesystem via `mount -t proc proc /proc`.

The proc filesystem then contains infos about the running processes, **as seen from the container**. And there is
[lots of helpful things](https://docs.kernel.org/filesystems/proc.html#process-specific-subdirectories) in the proc
filesystem (which I've only scratched the surface on): `/proc/[pid]/root` contains a **link to the root filesystem
of the given process**.

For "normal" processes (not running inside Linux namespaces), this will simply point to `/`. For processes in Linux
namespaces, it will **point to the root filesystem of the container**, even if it is not otherwise available :)

Thus, we simply run `ln -s /proc/1/root /container` - and this way, the container filesystem can be found
at `/container`.

### How do we get access to the Host machine in Docker OSX?

Because Docker relies on Linux namespaces (a Linux kernel feature), we cannot simply run a Docker container
on OSX. That's why Docker Desktop for Mac transparently creates a Linux VM, where then the Docker containers
are launched:

```
                          ┌────────────────────────────────┐
    Mac OS Host           │ Linux VM (mostly transparent)  │
╔══════════════════╗      │   ╔════════════════════════╗   │
║  Docker Client   ║──────┼──▶║     Docker Server      ║   │
╚══════════════════╝      │   ╚════════════════════════╝   │
                          │   ┌────────────────────────┐   │
                          │   │   Docker Containers    │   │
                          │   └────────────────────────┘   │
                          └────────────────────────────────┘
```

Remember we need get access to the **host machine** for running `nsenter`; so on OSX, it is the transparent
*Linux VM* we need to access - but we need to do this by running a Docker container (because that's the only
thing possible in the VM).

By running `docker run --privileged --pid=host`, we create a container that circumvents the default security
restrictions; **and which stays in the Host's process namespace**.

This way, if we run `ps -ef`, we get the processes of the host and of **all** running Docker containers.

We can also check out the Linux VM filesystem by inspecting `/proc/1/root`:

```
docker run --rm -it --privileged --pid=host ubuntu:kinetic /bin/bash

root@1f5429d2d626:/# ls /
bin  boot  dev  etc  home  lib  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var
root@1f5429d2d626:/# ls /proc/1/root/
bin  boot  containers  dev  etc  fakeowner.ko  grpcfuse.ko  home  init  lib  media  mnt  opt  proc  root  run  sbin  shiftfs.ko  srv  sys  tmp  usr  var
```

So you see the both directories differ: the first one is the file system of the `ubuntu:kinetic` image,
and `/proc/1/root` contains the root filesystem of the enclosing Linux VM.


## Web Mounting specials

(todo explain / write)

first try: embedded NFS server

second try: nfs-ganesha

third try: webdav

### Details about the `net` namespace

## Inspiration and resources

- [Docker CLI Plugins 1](https://gist.github.com/thaJeztah/b7950186212a49e91a806689e66b317d)
- [Docker CLI Plugins 2](https://dille.name/slides/2019-06-06/020_advanced/080_docker_cli_plugins/slides/)
- https://enqueuezero.com/container-and-nsenter.html
- https://collabnix.com/how-to-access-docker-namespace/
- https://dev.to/stefanjacobs/nsenter-entering-a-running-process-or-container-46a3
- https://man7.org/linux/man-pages/man1/nsenter.1.html
- https://github.com/nicolaka/netshoot


