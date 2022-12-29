# drydock

> Docker Debugging Tools

- `drydock execroot`: Like `docker exec`, but *always* spawn a root shell
- `drydock vscode`: Open Visual Studio Code (with the [Containers extension](https://aka.ms/vscode-remote/download/containers)),
  allowing to edit any file as root
- `drydock spx`: Install and enable the [SPX Profiler](https://github.com/NoiseByNorthwest/php-spx) PHP extension
  into a running container (without restart).

* **Portable** written in Golang with no external dependencies. Install by downloading a single binary.

[Installation](#installation)
[GitHub](https://github.com/sandstorm/drydock)
[Features](#documentation)
