# Tutorial

This document walks you through submitting your successful build on Djinn CI,
after which you should have a high-level overview as to how Djinn CI works.

* [Prerequisites](#prerequisites)
* [Your first build](#your-first-build)
* [Building code](#building-code)
* [Further reading](#further-reading)

## Prerequisites

To start using Djinn CI, you will need an account. You can either
[create]({{index .Vars "host"}}/register) an account, or
[login]({{index .Vars "host"}}/login) using [GitHub](https://github.com) or
[GitLab](https://gitlab.com).

## Your first build

Once logged in click the [Submit]({{index .Vars "host"}}/builds/create) button
on the homepage. From here you will be able to submit your first build to Djinn CI.

Build's in Djinn CI are described using YAML manifests. These manifests describe
how the build is executed, from the driver to use, the jobs to run, and the
artifacts to collector. So let's write a simple manifest to demonstrate this,

    driver:
      type: qemu
      image: debian/stable
    stages:
    - os
    jobs:
    - stage: os
      commands:
      - cat /etc/os-release
      artifacts:
      - /etc/os-release

From this same page we can also add a comment to the build, and some tags to
help search for it later on. Tags are simple a comma separated list of strings.

Submitting the manifest will redirect you to the newly submitted build. From
here you will be able to see the status of the build as it's processed by Djinn CI.

Once the build has finished processing its status will be updated to reflect
so. If successful the build will be marked as `passed`. From the "Artifacts"
tab of the build page you will be able to see the `/etc/os-release` artifact
we collected from the build.

## Building code

Now that we've submitted a simple build manifest let's submit another one, only
this time let's actually build some source code, and collect the artifacts from
it. For this build we will make use of the `docker` driver,

    driver:
      type: docker
      image: golang:latest
      workspace: /go
    sources:
    - https://github.com/andrewpillar/mdsrv
    stages:
    - make
    jobs:
    - stage: make
      commands:
      - cd mdsrv
      - ./make.sh
      artifacts:
      - mdsrv/bin/mdsrv

The above manifest will download the source for
[mdsrv](https://github.com/andrewpillar/mdsrv) build it, and collect the built
binary.

## Further reading

* [Builds](/user/builds) - Learn about what a build is, how they're defined and
how they're executed
* [Manifest](/user/manifest) - Learn about build manifests, and what each
property within a manifest is used for during build execution
* [Drivers](/user/drivers) - Learn about the different drivers that can be used
for executing builds
* [Namespaces](/user/namespaces) - Learn about how namespaces can be used for
grouping builds and their resources together
* [Repos](/user/repos) - Learn how to trigger builds from pushes to GitHub or
GitLab
