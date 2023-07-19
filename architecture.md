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
  * [Consumer](#consumer)
* [Entrypoints](#entrypoints)
* [Code overview](#code-overview)
  * [Assets](#assets)
* [Code map](#code-map)
  * [auth](#auth)
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

### Consumer

The consumer is another background worker that handles other long running jobs,
such as the downloading of custom build images from external endpoints. This
has no effect on running builds.

## Entrypoints

Detailed below are the entrypoints for each component. This briefly details the
codepaths that are taken to get the component started and ready for execution.

**`djinn-server`**

The main entrypoint for the server is `cmd/djinn-server/main.go`, which sets
up the server, the in memory queue for webhook dispatching, and registering of
routes. This all occurs in `serverutil/serverutil.go`.

**`djinn-worker`**

The main entrypoint for the worker is `cmd/djinn-worker/worker.go`, which
sets up the worker, the in memory queue for webhook dispatching, and the
primary mechanism for consuming from the queue. This all occurs in
`workerutil/workerutil.go`.

**`djinn-scheduler`**

The main entrypoint for the scheduler is `cmd/djinn-scheduler/main.go`. In here
a loop runs on a configured interval, during which the cron jobs are performed
in batches of the configured size, by default 1000. The batching of the job is
handled via the `cron.Batcher` in `cron/batcher.go`.

**`djinn-curator`**

The main entrypoint for the curator is `cmd/djinn-curator/main.go`. In here a
loop runs on a 1 minute interval (non-configurable), during which old artifacts
that exceed the limit for a user are removed. The removal of the artifacts is
handled by the `build.Curator` in `build/curator.go`.

**`djinn-consumer`**

The main entrypoint for the consumer is `cmd/djinn-consumer/mainn.go`, which
sets up the consumer for handling long running background jobs necessary for
the `djinn-server`.

## Code overview

The logic for Djinn CI is grouped on a responsibility basis. For example, the
logic for the HTTP handlers exist in the `http` directories depending on the
entity that logic is for. Within each of these directories will be an `api.go`
and `ui.go` file, which will contain the logic for handling requests to the
API server and UI server respectively, what with shared logic between the two
being in the `handler.go` file.

## Code map

Detailed below is how the code base is structured and how it's best navigated.

### auth

`auth` contains the logic for authentication throughout Djinn CI. This also
provides implementations for authentication from 3rd party providers via
OAuth2.

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

`config` is the package that handles the decoding of configuration files for
Djinn CI. Each configuration file uses a block-based configuration format,
where blocks of configuration are wrapped between `{ }`. The configuration
structs do no represent a one-to-one mapping of what's in the file, instead
they expose for necessary resources for a component to function.

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
This provides some custom types to make working with the database easier, and
makes heavy use of the [andrewpillar/query][0] library for query building.

[0]: https://github.com/andrewpillar/query

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
as an OAuth2 server.

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

### queue

`queue` provides an abstraction and some implementations for a simple queue.
This is mainly used as the mechanism by which webhooks are dispatched.

### template

`template` contains all of the templates used for rendering all of the HTML
pages that `djinn-server` serves.

[valyala/quicktemplate][1] is used for templating the views that `djinn-server`
serves via the UI. This allows for these views to be compiled directly into the
final binary itself.

[1]: https://github.com/valyala/quicktemplate

### version

`version` contains version information about `djinn-server`. This is set during
the linking stage of the build by the linker.

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
