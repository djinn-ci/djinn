[Prev](/admin/building) - [Next](/admin/worker)

# Server

* [External dependencies](#external-dependencies)
* [Configuring the server](#configuring-the-server)
* [Example server configuration](#example-server-configuration)
* [Running the server](#running-the-server)
* [Configuring the server daemon](#configuring-the-server-daemon)

## External dependencies

Detailed below are the software dependencies that the Djinn server in order to
start and run,

| Dependency  | Reason                                                    |
|-------------|-----------------------------------------------------------|
| PostgreSQL  | Primary data store for the server.                        |
| Redis       | Data store for session data, and used as the build queue. |

## Configuring the server

The server is configured via a `server.toml` file, detailed below are the
properties for this file,

* `host` - the host on which the server will be running on, this will be used
for OAuth redirects and setting the endpoint to which the webhooks are sent.
This can be either an IP address or a FQDN, though it is recommend to be the
latter.

* `net.listen` - The address that should be used to serve over.

* `net.ssl.cert` - The certificate to use if you want the server to serve over
TLS.

* `net.ssl.key` - The key to use if you want the server to serve over TLS.

* `crypto.hash` - The hash key to use for authenticating encrypted cookie
values via HMAC. This must be either 32, or 64 characters in length.

* `crypto.block` - The key to use for the block cipher that is used for
encrypting values. This must be either 16, 24, or 32 characters in length. This
should match what is configured for the worker.

* `crypto.salt` - The salt to use when generating secure unique hashes.

* `cypto.auth` - The key to use to protect against CSRF attacks. This must be 32
characters long.

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

* `objects.type` - The type of store to use for storing objects. Must be one of:
`file`.

* `objects.path` - The location of where to store objects to place into builds.

* `objects.limit` - The maximum size of objects that can be uploaded to the
server. Set to `0` for unlimited.

* `log.level` - The level of logging to use whilst the server is running. Must
be one of: `debug`, `info`, or `error`.

* `drivers.type` - The type of driver to support on the server. Must be one of:
`docker`, `qemu`, or `ssh`.

* `drivers.queue` - The name of the queue that builds for the given driver should
be submitted to.

* `providers.name` - The name of a Git provider to support integration to. Must
be one of: `github` or `gitlab`.

* `providers.secret` - The secret used to authenticated incoming webhooks from
the provider.

* `providers.client_id` - The `client_id` of the provider being integrated with.

* `providers.client_secret` - The `client_secret` of the provider being
integrated with.

## Example server configuration

An example `server.toml` file can be found in the `dist` directory of the
source repository.

    host = "https://localhost:8443"

    [net]
    listen = "localhost:8443"

    [net.ssl]
    cert = "/var/lib/ssl/server.crt"
    key  = "/var/lib/ssl/server.key"

    [crypto]
    hash  = "..."
    block = "..."
    salt  = "..."
    auth  = "..."

    [database]
    addr     = "localhost:5432"
    name     = "djinn"
    username = "djinn"
    password = "secret"

    [redis]
    addr = "localhost:6379"

    [images]
    type = "file"
    path = "/var/lib/djinn/images"

    [artifacts]
    type = "file"
    path = "/var/lib/djinn/artifacts"

    [objects]
    type  = "file"
    path  = "/var/lib/djinn/objects"
    limit = 5000000

    [log]
    level = "info"
    file  = "/var/log/djinn/server.log"

    [[drivers]]
    type  = "qemu"
    queue = "builds"

    [[providers]]
    name          = "github"
    secret        = "..."
    client_id     = "..."
    client_secret = "..."

    [[providers]]
    name          = "gitlab"
    secret        = "..."
    client_id     = "..."
    client_secret = "..."

## Running the server

To run the server simply invoke the `djinn-server` binary. There are three flags
that can be given to the `djinn-server` binary.

* `-config` - This specifies the configuration file to use, by default
this will be `djinn-server.toml`.

* `-api` - This tells the server to only serve the [REST API](/api) endpoints.

* `-ui` - This tells the server to only serve the UI endpoints.

If you do not specify either the `--api`, or `--ui` flag then both groups of
endpoints will be served. The [REST API](/api) endpoints will be served under
the `/api` prefix,

**Serving both the UI and API**

    $ djinn-server -c /etc/djinn/server.toml

**Serving just the API**

    $ djinn-server -c /etc/djinn/server.toml --api

**Serving just the UI**

    $ djinn-server -c /etc/djinn/server.toml --ui

## Configuring the server daemon

The `dist` directory contains files for running the Djinn server as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
