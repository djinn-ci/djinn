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

## Local development

You can download the `djinn-dev` image that is used in the build itself, and
use this for local development. The image can be downloaded [here][0]. This can
be used for local development, the machine can be booted by running the
`qemu.sh` script,

    $ ./qemu.sh

once booted you will be able to SSH into it as `root` on port `2222`,

    $ ssh -p 2222 root@localhost

the ports for Redis (`6379`), and PostgreSQL (`5432`) will also be locally
accessible too.

[mgrt][1] is used for performing revisions against for Djinn CI. Once the
virtual machine is booted, you can run the following mgrt command,

    $ mgrt db set djinn-dev postgresql "host=localhost port=5432 dbname=djinn user=djinn password=secret"
    $ mgrt run -c schema -db djinn-dev
    $ mgrt run -c perms -db djinn-dev
    $ mgrt run -c dev -db djinn-dev

[0]: https://djinn-ci.com/n/djinn-ci/djinn/-/images
[1]: https://github.com/andrewpillar/mgrt
