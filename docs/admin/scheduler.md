[Prev](/admin/curator) - [Next](/admin/server)

# Scheduler

The `djinn-scheduler` is the component that handles the scheduling of cron jobs
that have been submitted via the server. Every minute this will invoke the
cron jobs in batches of 1000 that are ready to be invoked.

* [External Dependencies](#external-dependencies)
* [Configuring the Scheduler](#configuring-the-scheduler)
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

Detailed below are the [configuration](/admin/configuration) directives used by
the scheduler.

* **`drivers`** `[...]`

The list of drivers supported on the server. This should match what is in the
[server configuration](/admin/server#configuring-the-server). This should only
contain string literals.

* **`database`** `{...}`

Provides connection information to the PostgreSQL database. Below are the
directives used by the `database` block directive.

* **`addr`** `string` - The address of the PostgreSQL server to connect to.

* **`name`** `string` - The name of the database to use.

* **`username`** `string` - The name of the database user.

* **`password`** `string` - The password of the database user.

* **`ssl`** `{...}` - SSL block directive if you want to connect via TLS.

  * **`ca`** `string` - Path to the CA root to use.
  * **`cert`** `string` - Path to the certificate to use.
  * **`key`** `string` - Path to the key to use.

* **`redis`** `{...}`

Provides connection information to the Redis database. Below are the directives
used by the `redis` block directive.

* **`addr`** `string` - The address of the Redis server to connect to.

* **`password`** `string` - The password used if the Redis server is
password protected.

## Example Scheduler Configuration

An example `scheduler.conf` file can be found in the `dist` directory of the
source repository.

    pidfile "/var/run/djinn/scheduler.pid"
    
    log info "/var/log/djinn/scheduler.log"
    
    drivers [
        "docker",
        "qemu-x86_64",
    ]
    
    database {
        addr "localhost:5432"
        name "djinn"
    
        username "djinn-scheduler"
        password "secret"
    }
    
    redis {
        addr "localhost:6379"
    }

## Running the Scheduler

To run the scheduler simply invoke the `djinn-scheduler` binary. There is only
one flag that can be given to the `djinn-scheduler` binary.

* `-config` - This specifies the configuration file to use, by default this
will be `djinn-scheduler.conf`.

## Configuring the Scheduler Daemon

The `dist` directory contains files for running the Djinn Scheduler as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
