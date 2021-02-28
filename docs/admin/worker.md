[Prev](/admin/server)

# Worker

The `djinn-worker` is the component that handles executing builds that are
submitted via the [server](/admin/server) or the [scheduler](/admin/scheduler).
You may need to install some additional dependencies on the worker machine
depending on the drivers you want to make available.

* [External Dependencies](#external-dependencies)
  * [Driver Dependencies](#driver-dependencies)
* [Configuring the Worker](#configuring-the-worker)
  * [General Configuration](#general-configuration)
  * [Crypto](#crypto)
  * [SMTP](#smtp)
  * [Database](#database)
  * [Redis](#redis)
  * [Images](#images)
  * [Artifacts](#artifacts)
  * [Objects](#objects)
  * [Logging](#logging)
* [Example Worker Configuration](#example-worker-configuration)
* [Running the Worker](#running-the-worker)
* [Configuring the Worker Daemon](#configuring-the-worker-daemon)

## External Dependencies

Detailed below are the software dependencies that the worker needs in order
to start and run,

| Dependency  | Reason                                                    |
|-------------|-----------------------------------------------------------|
| PostgreSQL  | Primary data store for the server.                        |
| Redis       | Data store for session data, and used as the build queue. |
| SMTP Server | Used for sending emails on build failures.                |

### Driver Dependencies

Detailed below are the software dependencies that the worker needs in order
to execute a build via that driver.

| Driver   | Software                                                  |
|----------|-----------------------------------------------------------|
| `docker` | The `dockerd` process for managing containers             |
| `qemu`   | The `qemu` software package for creating virtual machines |

## Configuring the worker

The server is configured via a `worker.toml` file, detailed below are the
properties for this file,

The worker also requires a `driver.toml` file to be configured, see details
on how to do this [here](/user/offline-runner#configuring-drivers).

### General Configuration

* `parallelism` - This specifies the parallelism to use when running multiple
builds at one. Set this to `0` to use the number of CPU cores available.

* `driver` - The driver we want to use when executing builds with the worker.
To use all drivers then set to `*`. For the `qemu` driver the arch must match
the host arch. For example, if running on `amd64` and you want to use the qemu
driver then you must specify `qemu-x86_64`.

* `timeout` - This specifies the duration after which builds should be killed.
Valid time units are `ns`, `us`, `ms`, `s`, `m`, and `h`.

### Crypto

* `crypto.block` - The key to use for the block cipher that is used for
encrypting values. This must be either 16, 24, or 32 characters in length. This
should match what is configured for the server.

### SMTP

* `smtp.addr` - The address of the SMTP server to use for sending mail.

* `smtp.ca` - The path to the CA root chain that should be used, if you want to
invoke the `STARTTLS` command on the SMTP server.

* `smtp.admin` - The postmaster's email. This will be used in the `From` field
in any mail that is sent via the SMTP server.

* `smtp.username` - The username of the account to authenticate with on the SMTP
server.

* `smtp.password` - The password of the account to authenticate with on the SMTP
server.

### Database

* `database.addr` - The address of the PostgreSQL server to connect to.

* `database.name` - The name of the database to use.

* `databse.username` - The name of the database user.

* `database.password` - The password of the database user.

### Redis

* `redis.addr` - The address of the Redis server to connect to.

### Images

* `images.type` - The type of store to use for storing custom image files. Must
be one of: `file`.

* `images.path` - The location of where custom image files are stored.

The directory specified in `images.path` must have a another sub-directory for
each supported driver. Within each of these directories should be a `_base`
directory to contain the base images to support.

For the QEMU driver within the `_base` directory should be another sub-directory
for each architecture. Within each of these should be the QEMU images to use.

### Artifacts

* `artifacts.type` - The type of store to use for storing artifacts. Must be one
of: `file`.

* `artifacts.path` - The location of where artifacts are stored.

* `artifacts.limit` - The maximum size of artifacts to collect from a build
environment. This will cut off collection of an individual artifact at the given
amount in bytes. Set to `0` for unlimited.

### Objects

* `objects.type` - The type of store to use for storing objects. Must be one of:
`file`.

* `objects.path` - The location of where to store objects to place into builds.

### Logging

* `log.level` - The level of logging to use whilst the server is running. Must
be one of: `debug`, `info`, or `error`.

* `log.file` - The file to write logs to, defaults to `/dev/stdout`.

## Example Worker Configuration

An example `worker.toml` file can be found in the `dist` directory of the
source repository.

    parallelism = 0

    queue = "builds"

    timeout = "30m"

    [crypto]
    block = "..."

    [database]
    addr     = "localhost:5432"
    name     = "djinn"
    username = "djinn-worker"
    password = "secret"

    [redis]
    addr = "localhost:6379"

    [images]
    type = "file"
    path = "/var/lib/djinn/images"

    [artifacts]
    type  = "file"
    path  = "/var/lib/djinn/artifacts"
    limit = 5000000

    [objects]
    type = "file"
    path = "/var/lib/djinn/objects"

    [log]
    level = "info"
    file  = "/var/log/djinn/worker.log"

## Running the Worker

To run the worker simply invoke the `djinn-worker` binary. There are two flags
that can be given to the `djinn-worker` binary.

* `-config` - This specifies the configuration file to use, by default
this will be `djinn-worker.toml`.

* `-driver` - This specifies the driver configuration file to use, for
configuring the drivers you want to support on your server, by default this
will be `djinn-driver.toml`.

## Configuring the Worker Daemon

The `dist` directory contains files for running the Djinn Worker as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
