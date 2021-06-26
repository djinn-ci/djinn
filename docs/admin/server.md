[Prev](/admin/scheduler) - [Next](/admin/worker)

# Server

The `djinn-server` is the primary component through which users would interact
with the CI server. This handles serving the UI and API endpoints for the
server. All assets served by the server are compiled into the binary itself,
so there is no need to worry about where assets will exist on disk.

* [External Dependencies](#external-dependencies)
* [Configuring the Server](#configuring-the-server)
* [Environment Variables](#environment-variables)
* [Example Server Configuration](#example-server-configuration)
* [Running the Server](#running-the-server)
* [Configuring the Server Daemon](#configuring-the-server-daemon)

## External Dependencies

Detailed below are the software dependencies that the server needs in order to
start and run,

| Dependency  | Reason                                                    |
|-------------|-----------------------------------------------------------|
| PostgreSQL  | Primary data store for the server.                        |
| Redis       | Data store for session data, and used as the build queue. |

## Configuring the Server

Detailed below are the [configuration](/admin/configuration) directives used by
the server.

* **`host`** `string`

The host on which the server will be running on, this will be used for OAuth
redirects and setting the endpoint to which the webhooks are sent. This can be
either an IP address or a FQDN, though it is recommend to be the latter.

* **`drivers`** `[...]`

The list of drivers you want to support on the server for the builds submitted.
This should match what is in the
[scheduler configuration](/admin/scheduler#configuring-the-scheduler). This
should only contain string literals.

* **`net`** `{...}`

Configuration details about how the server should be served over the network.

* **`listen`** `string` - The address that should be used to serve over.

* **`ssl`** `{...}` - The SSL block directive if you want to seve over TLS.

  * **`cert`** `string` - The path to the certificate to use.
  * **`key`** `string` - The path to the key to use.

* **`crypto`** `{...}`

Configuration settings for the cryptography used throughout the server for
encrypting data, generating hard to guess secrets, and protecting against CSRF
attacks.

* **`hash`** `string` -  The hash key is used to authenticate values using
HMAC. This must be either 32, or 64 characters in length.

* **`block`** `string` - The block key is required for encrypting data. This
must be either, 16, 24, or 32 characters in length.

* **`salt`** `string` -  Salt is used for generating hard to guess secrets,
and for generating the final key that is used for encrypting data.

* **`auth`** `string` -  The key to use to protect against CSRF attacks. This
must be 32 characters in length.

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

Configuration directives for each of the file stores the server uses. There must
be a store configured for each `artifacts`, `images`, and `objects`.

Detailed below are the value directives used within a `store` block directive.

* **`type`** `string` - The type of the store to use for the files being
accessed. Must be `file`.

* **`path`** `string` - The location of where the files are.

* **`limit`** `int` - The maximum size of files being uploaded. This will only
be applied to objects being uploaded to the server.

* **`provider`** `identifier` `{...}`

Configuration directives for each 3rd party provider you want the server to
integrate with. The `identifier` would be one of the supported providers
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

## Environment Variables

Detailed below are the environment variables that can be set for the Djinn CI
server,

* `DJINN_API_DOCS` - This is the link to the API documentation. This link is
rendered on the sidebar, if set.

* `DJINN_API_SERVER` - The host the API server is running on. This is used when
emitting webhook events, to ensure the links in the event payloads point back
to the API server.

* `DJINN_USER_DOCS` - This is the link to the user documentation. This link is
rendered on the sidebar, if set.

If deploying on a distribution with systemd, then it is recommended you put
these variables in an environment variable to be loaded in, for example,

    $ cat /etc/default/djinn
    DJINN_API_DOCS=https://docs.djinn-ci.com/api
    DJINN_API_SERVER=https://api.djinn-ci.com
    DJINN_USER_DOCS=https://docs.djinn-ci.com/user

## Example server configuration

An example `server.conf` file can be found in the `dist` directory of the
source repository.

    pidfile "/var/run/djinn/server.pid"

    log info "/var/log/djinn/curator.log"

    host "https://djinn-ci.com"

    drivers [
        "docker",
        "qemu-x86_64",
    ]

    net {
        listen ":443"

        ssl {
            cert "/var/lib/ssl/server.crt"
            key  "/var/lib/ssl/server.key"
        }
    }

    crypto {
        hash  "1a2b3c4d5e6f7g8h1a2b3c4d5e6f7g8h"
        block "1a2b3c4d5e6f7g8h"
        salt  "1a2b3c4d5e6f7g8h"
        auth  "1a2b3c4d5e6f7g8h1a2b3c4d5e6f7g8h"
    }

    database {
        addr "localhost:5432"
        name "djinn"

        username "djinn-server"
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
        type "file"
        path "/var/lib/djinn/artifacts"
    }

    store images {
        type "file"
        path "/var/lib/djinn/images"
    }

    store objects {
        type  "file"
        path  "/var/lib/djinn/objects"
        limit 5242880 
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

## Running the Server

To run the server simply invoke the `djinn-server` binary. There are three flags
that can be given to the `djinn-server` binary.

* `-config` - This specifies the configuration file to use, by default
this will be `djinn-server.conf`.

* `-api` - This tells the server to only serve the [REST API](/api) endpoints.

* `-ui` - This tells the server to only serve the UI endpoints.

If you do not specify either the `-api`, or `-ui` flag then both groups of
endpoints will be served. The [REST API](/api) endpoints will be served under
the `/api` prefix,

**Serving both the UI and API**

    $ djinn-server -config /etc/djinn/server.conf

**Serving just the API**

    $ djinn-server -config /etc/djinn/server.conf -api

**Serving just the UI**

    $ djinn-server -config /etc/djinn/server.conf -ui

## Configuring the Server Daemon

The `dist` directory contains files for running the Djinn Server as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
