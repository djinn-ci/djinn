[Prev](/admin/configuration) - [Next](/admin/scheduler)

# Curator

The `djinn-curator` is the component that handles cleaning up of old artifacts
that exceed a given limit. Every minute this will trigger and remove the oldest
artifacts that exceed the configured limit of the curator.

* [External Dependencies](#external-dependencies)
* [Configuring the Curator](#configuring-the-curator)
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

Detailed below are the [configuration](/admin/configuration) directives used by
the curator.

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

* **`store`** `identifier` `{...}`

The location where the build artifacts are stored. The `identifier` must be
`artifacts`.

* **`type`** `string` - The type of store to use, must be `file`.
* **`path`** `string` - The location of the artifacts.

## Example Curator Configuration

    pidfile "/var/run/djinn/curator.pid"

    log info "/var/log/djinn/curator.log"

    database {
        addr "localhost:5432"
        name "djinn"

        username "djinn-curator"
        password "secret"
    }

    store artifacts {
        type "file"
        path "/var/lib/djinn/artifacts"
    }

## Running the Curator

To run the curator simply invoke the `djinn-curator` binary. There are only two
flags that can be given to the `djinn-curator` binary,

* `-config` - This specifies the configuration file to use, by default this
will be `djinn-curator.conf`.

* `-limit` - This specifies the limit in bytes to use for clearing up
artifacts, by default this will be set to `1073741824` (1GB).

## Configuring the Curator Daemon

The `dist` directory contains files for running the Djinn Curator as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
