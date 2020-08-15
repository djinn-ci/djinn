[Prev](/admin/server) - [Next](/admin/deployments)

# Worker

* [External dependencies](#external-dependencies)
* [Configuring the worker](#configuring-the-worker)
* [Example worker configuration](#example-worker-configuration)
* [Running the worker](#running-the-worker)
* [Configuring the worker daemon](#configuring-the-worker-daemon)

## External dependencies

Detailed below are the software dependencies that the Djinn work in order to
start and run,

| Dependency  | Reason                                                    |
|-------------|-----------------------------------------------------------|
| PostgreSQL  | Primary data store for the server.                        |
| Redis       | Data store for session data, and used as the build queue. |
| SMTP Server | Used for sending emails on build failures.                |

## Configuring the worker

The server is configured via a `worker.toml` file, detailed below are the
properties for this file,

>**Note:** Under the hood the worker uses
[RichardKnop/machinery](https://github.com/RichardKnop/machinery) as its work
queue mechanism. This will write additional information about the builds being
processed to `stdout`.

* `webserver` - This is the address of the web server that serves Djinn CI. This
will be used for any links in emails.

* `parallelism` - This specifies the parallelism to use when running multiple
builds at one. Set this to `0` to use the number of CPU cores available.

* `queue` - This specifies the queue that builds should be popped off.

* `timeout` - This specifies the duration after which builds should be killed.
Valie time units are `ns`, `us`, `ms`, `s`, `m`, and `h`.

* `crypto.block` - The key to use for the block cipher that is used for
encrypting values. This must be either 16, 24, or 32 characters in length. This
should match what is configured for the server.

* `smtp.addr` - The address of the SMTP server to use for sending mail.

* `smtp.ca` - The path to the CA root chain that should be used, if you want to
invoke the `STARTTLS` command on the SMTP server.

* `smtp.admin` - The postmaster's email. This will be used in the `From` field
in any mail that is sent via the SMTP server.

* `smtp.username` - The username of the account to authenticate with on the SMTP
server.

* `smtp.password` - The password of the account to authenticate with on the SMTP
server.

* `database.addr` - The address of the PostgreSQL server to connect to.

* `database.name` - The name of the database to use.

* `databse.username` - The name of the database user.

* `database.password` - The password of the database user.

* `redis.addr` - The address of the Redis server to connect to.

* `images.type` - The type of store to use for storing custom image files. Must
be one of: `file`.

* `images.path` - The location of where custom image files are stored.

* `artifacts.type` - The type of store to use for storing artifacts. Must be one
of: `file`.

* `artifacts.path` - The location of where artifacts are stored.

* `artifacts.limit` - The maximum size of artifacts to collect from a build
environment. This will cut off collection of an individual artifact at the given
amount in bytes. Set to `0` for unlimited.

* `objects.type` - The type of store to use for storing objects. Must be one of:
`file`.

* `objects.path` - The location of where to store objects to place into builds.

* `log.level` - The level of logging to use whilst the server is running. Must
be one of: `debug`, `info`, or `error`.

## Example worker configuration

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

## Running the worker

To run the worker simply invoke the `djinn-worker` binary. There are two flags
that can be given to the `djinn-worker` binary.

* `-config` - This specifies the configuration file to use, by default
this will be `djinn-worker.toml`.

* `-driver` - This specifies the driver configuration file to use, for
configuring the drivers you want to support on your server, by default this
will be `djinn-driver.toml`.

## Configuring the worker daemon

The `dist` directory contains files for running the Djinn worker as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
