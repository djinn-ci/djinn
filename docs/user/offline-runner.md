[Prev](/user/keys) - [Next](/api)

# Offline Runner

Djinn CI can be used to run your builds offline without having to submit them to
the build server. This does require having to configure the necessary drivers
on your machine however.

* [Installing the Offline Runner](#installing-the-offline-runner)
* [Configuration Locations](#configuration-locations)
* [Configuring Drivers](#configuring-drivers)
  * [Docker](#docker)
  * [QEMU](#qemu)

## Installing the Offline Runner

To install the offline runner you will first need to build it using Go,

    $ git clone https://github.com/djinn-ci/djinn

once cloned, change into the directory and run the `make.sh` script.

    $ ./make.sh runner

This will produce a binary called `bin/djinn`, simply move this binary into a
location that will make it accessible via your `PATH`.

## Configuration Locations

For the offline runner, driver configuration sits in the `driver.toml` file. By
default this is expected to be in the user config directory. Detailed below is
where the file will be found on the different operating systems,

**Unix**

If non-empty then `$XDG_CONFIG_HOME` is used, and the fullpath would be
`$XDG_CONFIG_HOME/djinn/driver.toml`. Otherwise it will use
`~/.config/djinn/driver.toml`.

**Darwin**

On Darwin the path used will be,
`$HOME/Library/Application Support/djinn/driver.toml`.

**Windows**

On Windows the path used will be, `%AppData%/djinn`.

## Configuring Drivers

>**Note:** The same driver configuration used for the offline runner is used
for the worker too.

Each driver supported by Djinn CI is configured in its own block in the
`driver.toml` file, for example to configure the QEMU driver you would do the
following,

    [qemu]
    disks   = "/home/me/.config/djinn/images/qemu"
    cpus    = 1
    memory  = 2048

the above configuration would set the location of the QEMU disk images to use,
the number of CPUs, and the amount of memory for each machine that will be
created.

### Docker

The Docker driver is configured in the `[docker]` block of the `driver.toml`
configuration file. Detailed below are the different properties for this block,

* `host` -  The host of the running docker daemon, can be a path to a Unix
socket.

* `version` - The version of the Docker API you wish to use.

### QEMU

The QEMU driver is configured in the `[qemu]` block of the `driver.toml`
configuration file. Detailed below are the different properties for this block,

* `disks` - The location on the filesystem to look for the QEMU disk images to
use.

* `cpus` - The number of CPUs to use for a QEMU machine that is created for
execution.

* `memory` - The amount of memory in bytes for a QEMU machine that is created
for execution.

The directory specified in `disks` must have a another sub-directory for each
architecture, in each of these exist the disk images to use. For example assume
a manifest declares the following,

    driver:
      type: qemu
      image: centos/8

then Djinn will look for the following disk image,

    /home/me/.config/djinn/images/qemu/x86_64/centos/8
