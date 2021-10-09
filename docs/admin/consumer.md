[Prev](/admin/configuration) - [Next](/admin/curator)

# Consumer

The `djinn-consumer` is the component that handles the processing of background
jobs, such as remote image downloads. This will pull jobs from Redis off the
`jobs` queue for processing.

* [External Dependencies](#external-dependencies)
* [Configuring the Consumer](#configuring-the-consumer)
* [Example Consumer Configuration](#example-consumer-configuration)
* [Running the Consumer](#running-the-consumer)
* [Configuring the Consumer](#configuring-the-consumer)

## External Dependencies

Detailed below are the software dependencies that the curator needs in order
to start and run,

| Dependency | Reason                              |
|------------|-------------------------------------|
| PostgreSQL | Primary data store for the curator. |
| Redis      | Data store used as the build queue. |

## Configuring the Consumer

Detailed below are the [configuration](/admin/configuration) directives used by
the consumer.

* **`database`** `{...}` - Provides connection information to the PostgreSQL
database. Below are the directives used by the `database` block directive.

  * **`addr`** `string` - The address of the PostgreSQL server to connect to.
  * **`name`** `string` - The name of the database to use.
  * **`username`** `string` - The name of the database user.
  * **`password`** `string` - The password of the database user.

  * **`ssl`** `{...}` - SSL block directive if you want to connect via TLS.

    * **`ca`** `string` - Path to the CA root to use.
    * **`cert`** `string` - Path to the certificate to use.
    * **`key`** `string` - Path to the key to use.

* **`redis`** `{...}` - Provides connection information to the Redis database.
Below are the directives user by the `redis` block directive.

  * **`addr`** `string` - The address of the Redis server to connect to.
  * **`password`** `string` - The password used if the Redis server is
password protected.

* **`store`** `identifier` `{...}` - The location where the driver images are
stored. The `identifier` must be `images`.

  * **`type`** `string` - The type of store to use, must be `file`.
  * **`path`** `string` - The location of the artifacts.

## Example Consumer Configuration

    pidfile "/var/run/djinn/consumer.pid"

    log info "/var/log/djinn/consumer.log"

    database {
        addr "localhost:5432"
        name "djinn"
    
        username "djinn_curator"
        password "secret"
    }

    redis {
        addr "localhost:6379"
    }

    store images {
        type "file"
        path "/var/lib/djinn/images"
    }

## Running the Consumer

To run the c onsumer simply invoke the `djinn-consumer` b inary. There is only
one flag that can be given to the `djinn-consumer` binary,

* `-config` - This specifies the configuration file to use, by default this
will be `djinn-consumer.conf`.

## Configuring the Consumer Daemon

The `dist` directory contains files for running the Djinn Consumer as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
