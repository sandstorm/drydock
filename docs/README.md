# drydock - Docker Debugging Tools

This is a set of useful CLI tools to help debugging Docker containers:

**General Tooling**

- `drydock execroot`: Like `docker exec`, but *always* spawn a root shell
- `drydock vscode`: Open Visual Studio Code (with the [Containers extension](https://aka.ms/vscode-remote/download/containers)),
  allowing to edit any file as root

**PHP Specific Tooling**

- `drydock spx`: Install and enable the [SPX Profiler](https://github.com/NoiseByNorthwest/php-spx) PHP extension
  into a running container (without restart).
- `drydock xdebug`: Install and enable the [Xdebug](https://xdebug.org) PHP extension
  into a running container (without restart). Additionally, supports mounting extra folders over webdav for a better OSX
  debugging experience.

## Installation

We have tested the tools in OSX. They also can work on other platforms, but were not tested there yet.

1. Run the following command to install via homebrew (OSX):

   ```bash
   brew install sandstorm/tap/drydock
   ```

2. Done :)

   ```bash
   drydock --help  
   ```

## Documentation

**click the links for the full documentation for each command**

* [`drydock execroot [containername]`](https://sandstorm.github.io/drydock/#execroot)
* [`drydock vscode [containername]`](https://sandstorm.github.io/drydock/#vscode)
* [`drydock spx [containername]`](https://sandstorm.github.io/drydock/#spx)
* [`drydock xdebug [containername]`](https://sandstorm.github.io/drydock/#xdebug)


## License

MIT
