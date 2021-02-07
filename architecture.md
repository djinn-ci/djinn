# Architecture

This document details the architecture for Djinn CI. This will cover everything
from a high-level overview as to how the overall system fits together, to the
layout of the source codes files, and the various entrypoints for the different
components of the system.

* [30,000ft view](#30000ft-view)
  * [Server](#server)
  * [Worker](#worker)
  * [Scheduler](#scheduler)
  * [Curator](#curator)
* [Entrypoints](#entrypoints)
* [Code overview](#code-overview)
  * [Assets](#assets)
* [Code map](#code-map)
  * [cmd](#cmd)
  * [config](#config)
  * [crypto](#crypto)
  * [database](#database)
  * [driver](#driver)
  * [errors](#errors)
  * [integration](#integration)
  * [log](#log)
  * [mail](#mail)
  * [oauth2](#oauth2)
  * [provider](#provider)
  * [runner](#runner)
  * [server](#server)
  * [serverutil](#serverutil)
  * [template](#template)
  * [version](#version)
  * [web](#web)
  * [worker](#worker)
  * [workerutil](#workerutil)

This document is for people who would like to get an overview as to how Djinn
CI works. This will cover everything from the server itself to the worker, and
the offline runner.

## 30,000ft view

At the highest level Djinn CI takes builds that are submitted to the server and
places them on a queue for the worker to execute when they're retrieved. The
worker handles preparing the build environment, running the build manifest, and
collecting build artifacts. Below is a diagram demonstrating this.

                                  +------------------+    +------------------+
                                  |                  |    |                  |
                               +--+  djinn-scheduler |    |  djinn-curator   +------+
                               |  |                  |    |                  |      |
             +                 |  +--------^---------+    +---------^--------+      |
             |                 |           |                        |               |
             |                 |      Poll for cron           Poll for artifacts    |
          Manifest             |           |                        |               |
             |                 |           +---------+--------------+               |
             |                 |                     |                              |
    +--------v---------+       |                     |                              |
    |                  |       |                     |                        Remove old
    |  djinn-server    +----------------------+      |                        build artifacts
    |                  |       |              |      |                              |
    +--------+---------+       |          Build info/|                              |
             |             Build job      cron jobs  |                              |
             |                 |              |      |                              |
         Build job             |              |      |                              |
             |           +-----v------+    +--v------+--+    +--------------+       |
             |           |            |    |            |    |              |       |
             +----------->  Queue     |    |  Database  |    |  File Store  <-------+
                         |            |    |            |    |              |
                         +-----+------+    +-----^------+    +---^--------+-+
                               |                 |               |        |
                               |                 |               |        |
                               |                 |         Collect build  |
                           Build job           Status      artifacts      |
                               |                 |               |        |
                               |                 |               |        |
                               |                 |               |    Place build
                               |       +---------+--------+      |    objects
                               |       |                  +------+        |
                               +------->  djinn-worker    |               |
                                       |                  <---------------+
                                       +------------------+


### Server

The server provides two interfaces to interacting with the CI system, a REST
API and an HTML web view. Through this interface you can submit build
manifests, create cron jobs, and hook into external services for automated
builds. Each build that is created will be submitted onto the queue for
processing via the worker. Information about the build (artifacts, objects,
variables, keys, etc.) are stored in the database upon build creation.

### Worker

The worker is what pulls build jobs off the queue and executes them based off
of what was described in the build manifest. During build execution this will
update the database with the new build information, such as its status, output
and progress. The worker will pull objects from the file store to place into
the build environment for use within the build, and collects artifacts from the
build environment to put into the file store.

### Scheduler

The scheduler will poll the database for any cron jobs that are sheduled to be
run. Each job that is scheduled to be run will have a build created for it,
and submitted to the queue. By default the scheduler will group the jobs to be
scheduled into batches of 1000.

### Curator

The curator will poll the database for any artifacts that are old and exceed
the storage space of 1GB. Any artifact that does exceed this threshold will
be removed from the file store. This will only clean up artifacts if a user has
this configured via their account settings.

## Entrypoints

Detailed below are the entrypoints for each component. This briefly details the
codepaths that are taken to get the component started and ready for execution.

**`djinn-server`**

From `cmd/djinn-server/main.go` a call to `serverutil.ParseFlags` is made, this
will parse the flags given to the `djinn-server` binary and returns the result
of these flags. The path to the configuration file to use is then given to
`serverutil.Init`, from here the configuration is fully initialized and used to
register the different entity routers against the underlying HTTP server. Some
additional handlers are also registered for handling 404 and 405 responses.
Once the routes have been registered the server is returned along with a
function for cleaning up the resources being used by the server. We then begin
serving requests and wait til a cancellation signal is received.

**`djinn-worker`**

From `cmd/djinn-worker/main.go` a call to `workerutil.ParseFlags` is made, this
will parse the flags given to the `djinn-worker` binary and returns the result
of these flags. The path to the configuration and driver file to use are then
given to `workerutil.Init`, from here the configuration for the worker and
driver's are fully initialized. We return the fully configuration
`worker.Worker`, this is then passed to `workerutil.Start` which starts the
worker in a goroutine. We then wait til a cancellation signal is received.

**`djinn-scheduler`**

From `cmd/djinn-scheduler/main.go` we parse the program's flags, and
initialize configuration for the scheduler. A ticker is then created that
ticks on a 1 minute interval. Every time this interval passes we load in the
cron jobs to be scheduled via the `cron.Batcher` and have them invoked. We then
do this continuously until a cancellation signal is received.

**`djinn-curator`**

From `cmd/djinn-curator/main.go` we parse the program's flags, and initialize
configuration for the curator. A ticker is then created that ticks on a 1
minute interval. Every time this interval passes we load in the old artifacts
to clear our via the `build.Curator` and delete them from disk. We then do this
continuously until a cancellation signal is received.

## Code overview

The logic for an entity within Djinn CI is grouped within its own directory,
along with the handler, router, and templates for said entity. For example
the logic for handling the creation, and viewing of builds is within the
`build` directory, what with the router existing in `build/router` and the
handler in `build/handler`.

Most entities will have an `api.go`, and `ui.go` file in their `handler`
directory. These are the handlers that would handle API and UI requests
from the server respectively. The core logic for an entity is stored in
the entity handler itself. For example the `build` handler exists in
`build/handler/build.go`, this is then embedded by the `api.go` and `ui.go`
handlers.

[valyala/quicktemplate](https://github.com/valyala/quicktemplate) is used for
templating the views that `djinn-server` serves via the UI. This allows for
these views to be compiled directly into the final binary itself.

## Code map

Detailed below is how the code base is structured and how it's best navigated.

### cmd

`cmd` contains the main entrypoints for each of the components in Djinn CI.
Each of these contain a single `main.go` file that bootstraps the necessary
component for execution.

* `cmd/djinn` - The offline runner
* `cmd/djinn-curator` - The curator
* `cmd/djinn-scheduler` - The scheduler
* `cmd/djinn-server` - The server
* `cmd/djinn-worker` - The worker

### config

`config` is the package that handles the parsing and unmarshalling of the
various configuration files for Djinn CI. Each configuration file is a TOML
file. The configuration structs for each component do not represent a
one-to-one mapping of what's in the file, instead they are structs that contain
the necessary resources for that component to function.

For example, the `config.Server` struct will contain an underlying database
connection to the database that is being connected to. These config structs
are typically passed around the program during the bootstrapping phase.

### crypto

`crypto` is the package that contains utility functions and structs for
handling the encryption/decryption of data, and for the generation of unique
hashes.

### database

`database` provides a basic interface for modelling data from the database,
along with utility functions for working with relationships between entities.

`djinn-server` makes heavy use of the `Loader` interface during bootstrapping
of an entity's router to define which relationships should be loaded onto an
entity during querying.

The `database.Store` struct that is exported offers a low-level way of handling
common CRUD operations on a table. This makes heavy use of the
[andrewpillar/query](https://github.com/andrewpillar/query) library for query
building.

### driver

`driver` provides the implementations of the `runner.Driver` interface. This is
what allows for a build to be executed in either a Docker container, or a QEMU
virtual machine. Within this directory is a sub-directory for each
implementation of a driver, for example `driver/qemu` contains the QEMU driver
implementation.

### errors

`errors` provides utility functions for error reporting. The function
`errors.Err` is heavily used for providing additional stacktracing to errors
that are raised.

### integration

`integration` contains the integration tests that are used to test the API of
Djinn CI. These will only run of the build tag `integration` is given when
running the tests.

### log

`log` provides a simple logging mechanism for logging messages at different
levels.

### mail

`mail` provides a simple SMTP client for sending plain-text emails. This is
used by the `djinn-server` for sending emails for account verification, and by
the `djinn-worker` to alert of failed builds.

### oauth2

`oauth2` provides the implementations required for the `djinn-server` to act
as an OAuth2 server. The routes for OAuth2 authentication are registered via
the router in `oauth2/router`. The handlers themselves exist in
`oauth2/handler`, and the templates exists in `oauth2/template`.

### provider

`provider` provides the implementations required for the `djinn-server` to act
as an OAuth2 client to the providers that can be integrated with. There is a
sub-directory for each provider that we do integrate with, for example
`provider/github` for GitHub.

### runner

`runner` is the package that allows for arbitrary jobs to be run via the
`runner.Driver` interface it exports. This also exports the `runner.Placer`
interface to allow for objects to be placed into the environment in which the
jobs are run, and exports the `runner.Collector` interface to allow for
artifacts to be collected. This is what the `djinn-worker` and `djinn` offline
runner use to handle execution of build manifests.

### server

`server` provides an HTTP server implementation that wraps the `http.Server`
from the standard library. This provies a way of easily registering groups of
routes against the server via the `server.Router` interface. This also provides
a way of serving the UI and API servers as two distinct things.

### serverutil

`serverutil` provides an interface for easily bootstrapping the server used
for serving the API and UI routes in `djinn-server`. This handles the parsing
of the flags that are passed to the `djinn-server` binary, and the
initialization of the server's configuration. This will also register all
of the routers for the entities that the server manages.

### template

`template` contains the base templates that are used for rendering the dashboard
views that `djinn-server` serves. This also provides templating for HTML forms
too.

### version

`version` contains version information about `djinn-server`. This is set during
the linking stage of the build by the linker.

### web

`web` provides utility functions for working with HTTP requests, and responses.
This also exports the `web.Handler` struct that is used throughout the
`djinn-server` and provides an easy way of getting the current user from an
incoming request. The `web.Middleware` provides middleware for checking for
guest/authenticated users, and provides a way of determining the amount of
access user has to an endpoint via the use of `web.Gate`.

### worker

`worker` contains the logic of the `djinn-worker`. This is what handles the
actual execution of the builds that are retrieved from the queue. This will
handle the placement of objects into the build environment, and the collection
of artifacts. This also keeps the database up to date with the status of the
build as it runs, and fires off any emails should a build fail.

### workerutil

`workerutil` provides an interface for easily bootstrapping the worker for
handling build execution. Similar to `serverutil`, this will handle the parsing
of flags that are passed to the `djinn-worker` binary, and the initialization
of the worker's configuration.
