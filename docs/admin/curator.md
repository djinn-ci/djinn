[Prev](/admin/building) - [Next](/admin/scheduler)

# Curator

The `djinn-curator` is the component that handles cleaning up of old artifacts
that exceed a given limit. Every minute this will trigger and remove the oldest
artifacts that exceed the configured limit of the curator.

* [External Dependencies](#external-dependencies)
* [Configuring the Curator](#configuring-the-curator)
  * [Database](#database)
  * [Artifacts](#artifacts)
  * [Logging](#logging)
* [Example Curator Configuration](#example-server-configuration)
* [Running the Curator](#running-the-curator)
* [Configuring the Curator Daemon](#configuring-the-curator-daemon)

## External Dependencies

Detailed below are the software dependencies that the curator needs in order
to start and run,

| Dependency  | Reason                              |
|-------------|-------------------------------------|
| PostgreSQL  | Primary data store for the curator. |

## Configuring the Curator

The curator is configured via a `curator.toml` file, detailed below are the
properties for this file.

### Database

* `database.addr` - The address of the PostgreSQL server to connect to.

* `database.name` - The name of the database to use.

* `databse.username` - The name of the database user.

* `database.password` - The password of the database user.

### Artifacts

* `artifacts.type` - The type of store to use for storing artifacts. Must be one
of: `file`.

* `artifacts.path` - The location of where artifacts are stored.

### Logging

* `log.level` - The level of logging to use whilst the server is running. Must
be one of: `debug`, `info`, or `error`.

* `log.file` - The file to write logs to, defaults to `/dev/stdout`.

## Example Curator Configuration

    [database]
    addr     = "localhost:5432"
    name     = "djinn"
    username = "djinn"
    password = "secret"
    
    # Where the artifacts to clean are stored.
    [artifacts]
    type = "file"
    path = "/var/lib/djinn/artifacts"
    
    [log]
    level = "debug"
    file  = "/dev/stdout"

## Running the Curator

To run the curator simply invoke the `djinn-curator` binary. There are only two
flags that can be given to the `djinn-curator` binary,

* `-config` - This specifies the configuration file to use, by default this
will be `djinn-curator.toml`.

* `-limit` - This specifies the limit in bytes to use for clearing up
artifacts, by default this will be set to "1GB".

## Configuring the Curator Daemon

The `dist` directory contains files for running the Djinn Curator as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
