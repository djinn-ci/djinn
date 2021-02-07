# Djinn CI

Djinn CI is a continuous integration system that allows for running builds in
Docker containers and Linux VMs. Builds can be run on the server, or they can
be run offline using the offline runner. Each build is configured via a simple
YAML manifest that describes how the build should be run, and what commands
should be executed within the build.

## Contributing

Before you start contributing read the [Architecture](architecture.md) document
that details the architecture of Djinn CI and how everything fits together.

## Building from source

If you cannot get hold of a binary distribution then you can always build Djinn
from source. Read through [docs/admin/building.md](docs/admin/building.md) for
more information on building from source.
