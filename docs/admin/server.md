[Prev](/admin/scheduler) - [Next](/admin/worker)

# Server

The `djinn-server` is the primary component through which users would interact
with the CI server. This handles serving the UI and API endpoints for the
server. All assets served by the server are compiled into the binary itself,
so there is no need to worry about where assets will exist on disk.

* [External Dependencies](#external-dependencies)
* [Configuring the Server](#configuring-the-server)
  * [Network](#network)
  * [Crypto](#crypto)
  * [Database](#database)
  * [Redis](#redis)
  * [Images](#images)
  * [Artifacts](#artifacts)
  * [Objects](#objects)
  * [Logging](#logging)
  * [Drivers](#drivers)
  * [Providers](#providers)
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

The server is configured via a `server.toml` file, detailed below are the
properties for this file,

* `host` - the host on which the server will be running on, this will be used
for OAuth redirects and setting the endpoint to which the webhooks are sent.
This can be either an IP address or a FQDN, though it is recommend to be the
latter.

### Network

* `net.listen` - The address that should be used to serve over.

* `net.ssl.cert` - The certificate to use if you want the server to serve over
TLS.

* `net.ssl.key` - The key to use if you want the server to serve over TLS.

### Crypto

* `crypto.hash` - The hash key to use for authenticating encrypted cookie
values via HMAC. This must be either 32, or 64 characters in length.

* `crypto.block` - The key to use for the block cipher that is used for
encrypting values. This must be either 16, 24, or 32 characters in length. This
should match what is configured for the worker.

* `crypto.salt` - The salt to use when generating secure unique hashes.

* `cypto.auth` - The key to use to protect against CSRF attacks. This must be 32
characters long.

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

### Artifacts

* `artifacts.type` - The type of store to use for storing artifacts. Must be one
of: `file`.

* `artifacts.path` - The location of where artifacts are stored.

### Objects

* `objects.type` - The type of store to use for storing objects. Must be one of:
`file`.

* `objects.path` - The location of where to store objects to place into builds.

* `objects.limit` - The maximum size of objects that can be uploaded to the
server. Set to `0` for unlimited.

### Logging

* `log.level` - The level of logging to use whilst the server is running. Must
be one of: `debug`, `info`, or `error`.

* `log.file` - The file to write logs to, defaults to `/dev/stdout`.

### Drivers

The `[[drivers]]` table specifies the drivers that are provided by Djinn for
executing builds. This expects the `type` of driver available, and the `queue`
to place the builds on. It is valid for different driver types to be placed on
to the same queue.

* `drivers.type` - The type of driver to support on the server. Must be one of:
`docker`, `qemu`, or `ssh`.

* `drivers.queue` - The name of the queue that builds for the given driver should
be submitted to.

### Providers

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

## Running the Server

To run the server simply invoke the `djinn-server` binary. There are three flags
that can be given to the `djinn-server` binary.

* `-config` - This specifies the configuration file to use, by default
this will be `djinn-server.toml`.

* `-api` - This tells the server to only serve the [REST API](/api) endpoints.

* `-ui` - This tells the server to only serve the UI endpoints.

If you do not specify either the `-api`, or `-ui` flag then both groups of
endpoints will be served. The [REST API](/api) endpoints will be served under
the `/api` prefix,

**Serving both the UI and API**

    $ djinn-server -config /etc/djinn/server.toml

**Serving just the API**

    $ djinn-server -config /etc/djinn/server.toml -api

**Serving just the UI**

    $ djinn-server -config /etc/djinn/server.toml -ui

## Configuring the Server Daemon

The `dist` directory contains files for running the Djinn server as a daemon
on Linux systems that use systemd and SysVinit for daemon management. Use
whichever suits your needs, and modify accordingly.

If deploying to a Linux system that uses systemd, then be sure to run
`systemctl daemon-reload` upon placement of the service file.
