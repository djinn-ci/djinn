[Prev](/admin/server)

# Worker

The `djinn-worker` is the component that handles executing builds that are
submitted via the [server](/admin/server) or the [scheduler](/admin/scheduler).
You may need to install some additional dependencies on the worker machine
depending on the drivers you want to make available.

* [External Dependencies](#external-dependencies)
  * [Driver Dependencies](#driver-dependencies)
* [Configuring the Worker](#configuring-the-worker)
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

Detailed below are the [configuration](/admin/configuration) directives used by
the worker. The worker also requires a `driver.conf` file to be configured, see
details on how to do this [here](/user/offline-runner#configuring-drivers).

* **`parallelism`** `int`

This specifies the parallelism to use when running multiple builds at one. Set
this to `0` to use the number of CPU cores available.

* **`driver`** `string`

The driver we want to use when executing builds with the worker. To use all
drivers then set to `*`. For the `qemu` driver the arch must match the host
arch. For example, if running on amd64 and you want to use the qemu driver then
you must specify `qemu-x86_64`.

* **`timeout`** `string`

The duration after which builds should be killed. Valid time units are `ns`,
`us`, `ms`, `s`, `m`, `h`.

* **`crypto`** `{...}`

Configuration settings for decrypting of data. The value directives used here
should match what was put in the
[server configuration](/admin/server#configuring-the-server).

* **`block`** `string` - The block key is required for encrypting data. This
must be either, 16, 24, or 32 characters in length.

* **`salt`** `string` -  Salt is used for generating hard to guess secrets,
and for generating the final key that is used for encrypting data.

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

* **`smtp`** `{...}`

Provides connection information to an SMTP server to sending emails. Below are
the directives used by the `smtp` block directive.

* **`addr`** `string` - The address of the SMTP server.

* **`ca`** `string` - If connecting via TLS, then the path to the file that
contains a set of root certificate authorities.

* **`admin`** `string` - The email address to be used in the `From` field of
mails that are sent.

* **`username`** `string` - The username for authentication.

* **`password`** `string` - The password for authentication.

* **`store`** `identifier` `{...}`

Configuration directives for each of the file stores the worker uses. There must
be a store configured for each `artifacts`, `images`, and `objects`.

Detailed below are the value directives used within a `store` block directive.

* **`type`** `string` - The type of the store to use for the files being
accessed. Must be `file`.

* **`path`** `string` - The location of where the files are.

* **`limit`** `int` - The maximum size of files being uploaded. This will only
be applied to artifacts collected from builds.

* **`provider`** `identifier` `{...}`

Configuration directives for each 3rd party provider we integrate with. These
are used to handle updating commit statuses if a build was triggered via
a pull request. The `identifier` would be one of the supported providers
detailed below,

* `github`
* `gitlab`

Detailed below are the value directives used within a `provider` block
directive.

* **`secret`** `string` - The secret used to authenticate incoming webhooks
from the provider.

* **`client_id`** `string` - The `client_id` of the provider being integrated
with.

* **`client_secret`** `string` - The `client_secret` of the provider being
integrated with.

## Example Worker Configuration

An example `worker.conf` file can be found in the `dist` directory of the
source repository.

    pidfile "/var/run/djinn/worker.pid"

    log info "/var/log/djinn/worker.log"

    parallelism 0

    driver "*"

    timeout "30m"

    crypto {
        block "1a2b3c4d5e6f7g8h"
        salt  "1a2b3c4d5e6f7g8h"
    }

    database {
        addr "localhost:5432"
        name "djinn"

        username "djinn-worker"
        password "secret"
    }

    redis {
        addr "localhost:6379"
    }

    smtp {
        addr "smtp.example.com:587"

        ca "/etc/ssl/cert.pem"

        admin "no-reply@djinn-ci.com"

        username "postmaster@example.com"
        password "secret"
    }

    store artifacts {
        type  "file"
        path  "/var/lib/djinn/artifacts"
        limit 52428800
    }

    store images {
        type "file"
        path "/var/lib/djinn/images"
    }

    store objects {
        type "file"
        path "/var/lib/djinn/objects"
    }

    provider github {
        secret "123456"

        client_id     "..."
        client_secret "..."
    }

    provider gitlab {
        secret "123456"

        client_id     "..."
        client_secret "..."
    }

## Running the Worker

To run the worker simply invoke the `djinn-worker` binary. There are two flags
that can be given to the `djinn-worker` binary.

* `-config` - This specifies the configuration file to use, by default
this will be `djinn-worker.conf`.

* `-driver` - This specifies the driver configuration file to use, for
configuring the drivers you want to support on your server, by default this
will be `djinn-driver.conf`.

## Configuring the Worker Daemon

The `dist` directory contains files for running the Djinn Worker as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
