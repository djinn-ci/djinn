[Prev](/admin/curator) - [Next](/admin/server)

# Scheduler

The `djinn-scheduler` is the component that handles the scheduling of cron jobs
that have been submitted via the server. Every minute this will invoke the
cron jobs in batches of 1000 that are ready to be invoked.

* [External Dependencies](#external-dependencies)
* [Configuring the Scheduler](#configuring-the-scheduler)
  * [Database](#database)
  * [Redis](#redis)
  * [Logging](#logging)
  * [Drivers](#drivers)
* [Example Scheduler Configuration](#example-server-configuration)
* [Running the Scheduler](#running-the-scheduler)
* [Configuring the Scheduler Daemon](#configuring-the-scheduler-daemon)

## External Dependencies

Detailed below are the software dependencies that the scheduler needs in order
to start and run,

| Dependency  | Reason                                                    |
|-------------|-----------------------------------------------------------|
| PostgreSQL  | Primary data store for the scheduler.                     |
| Redis       | Data store for session data, and used as the build queue. |

## Configuring the Scheduler

The scheduler is configured via a `scheduler.toml` file, detailed below are the
properties for this file,

### Database

* `database.addr` - The address of the PostgreSQL server to connect to.

* `database.name` - The name of the database to use.

* `databse.username` - The name of the database user.

* `database.password` - The password of the database user.

### Redis

* `redis.addr` - The address of the Redis server to connect to.

### Logging

* `log.level` - The level of logging to use whilst the server is running. Must
be one of: `debug`, `info`, or `error`.

* `log.file` - The file to write logs to, defaults to `/dev/stdout`.

### Drivers

The `[[drivers]]` table specifies the drivers that are provided by Djinn CI for
executing builds. This expects the `type` of driver available, and the `queue`
to place the builds on. It is valid for different driver types to be placed on
to the same queue.

* `drivers.type` - The type of driver to support on the server. Must be one of:
`docker`, `qemu`, or `ssh`. This should match what is being used by the
[server](/admin/server#drivers).

* `drivers.queue` - The name of the queue that builds for the given driver should
be submitted to. This should match what is being used by the
[server](/admin/server#drivers).

## Example Scheduler Configuration

An example `scheduler.toml` file can be found in the `dist` directory of the
source repository.

    [database]
    addr     = "localhost:5432"
    name     = "djinn"
    username = "djinn"
    password = "secret"
    
    [redis]
    addr = "localhost:6379"
    
    [log]
    level = "debug"
    file  = "/dev/stdout"
    
    [[drivers]]
    type  = "qemu"
    queue = "builds"
    
    [[drivers]]
    type  = "docker"
    queue = "builds"

## Running the Scheduler

To run the scheduler simply invoke the `djinn-scheduler` binary. There is only
one flag that can be given to the `djinn-scheduler` binary.

* `-config` - This specifies the configuration file to use, by default this
will be `djinn-scheduler.toml`.

## Configuring the Scheduler Daemon

The `dist` directory contains files for running the Djinn Scheduler as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
