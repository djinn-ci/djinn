[Prev](/user/drivers) - [Next](/user/images)

# Manifest

The build manifest is a YAML file that describes how a build should be executed.
Detailed below are the different fields for the manifest, what they do, and if
they are required.

* [Namespace](#namespace)
* [Driver](#driver)
* [Env](#env)
* [Objects](#objects)
* [Sources](#sources)
* [Stages](#stages)
* [Allow Failures](#allow-failures)
* [Jobs](#jobs)
  * [Name](#name)
  * [Stage](#stage)
  * [Commands](#commands)
  * [Artifacts](#artifacts)
* [Example manifest](#example-manifest)

## Namespace

**Required:** No

Specify the namespace to submit the build to. If the given namespace does not
exist then it will be created on the fly, and have the visibility of it set to
`private` by default. You can use the `<path>@<user>` notation to submit a build
to a namespace that you are a collaborator in.

**Standalone namespace**

    namespace: project

**Child namespace**

    namespace: project/child

**Namespace with owner**

    namespace: owner@project

**Namespace with owner and child**

    namespace: owner@project/child

## Driver

**Required:** Yes

Specify the driver to use for the build. All drivers require the `driver.type`
property. Each individual driver may have different requirements for each
subsequent property, more detail about the driver configuration can be found
in the [Drivers](/user/drivers) section of the user docs.

**QEMU driver**

    driver:
        type: qemu
        image: centos/7

**Docker driver**

    driver:
        type: docker
        image: golang
        workspace: /go

## Env

**Required:** No

Specify the environment variables to set during build execution, this expects
a list of strings formatted like so, `<key>=<value>`,

    env:
    - PGADDR=host=localhost port=5432 dbname=djinn user=djinn password=secret sslmode=disable
    - EDITOR=ed
    - LOCALE=en_GB.UTF-8

## Objects

**Required:** No

Specify the objects that you want placed into the build environment during
driver creation. This expects a list of strings, where each item is the name
of the uploaded object. The `=>` notation can be used to specify the full
destination location in the build environment the object should be placed in.

    objects:
    - data => /var/lib/data
    - keys.jks

>**Note:** Build times will increase depending on the number of objects being
placed into an environment and their size.

## Sources

**Required:** No

Specify the list of source code repositories to clone into the build
environment. Any repository URL recognized by `git clone` can be used here,

    sources:
    - https://github.com/andrewpillar/mdsrv.git
    - git@github.com:andrewpillar/mgrt.git

The destination name of the repository to clone can be set via the `=>`
notation,

    sources:
    - https://github.com/andrewpillar/mdsrv.git => mdsrv

the ref to checkout once cloned can be specified at the end of the URL.

    sources:
    - https://github.com/andrewpillar/mdsrv.git v1.0.0 => mdsrv

If no ref is specified then `master` is used as the default ref to checkout.

The sources in the manifest will be collated into a single job of the build
when the build is submitted. This means if any of the sources fail to clone then
the build itself will fail.

## Stages

**Required:** Yes

Specify the order in which stages should be executed.

    stages:
    - test
    - build

## Allow Failures

**Required:** No

Specify which stages are allowed to fail.

    allow_failures:
    - test

## Jobs

**Required:** Yes

Specify the jobs for the build to run. Each job will be executed in the order
in which it is specified.

    jobs:
    - stage: build
      commands:
      - go build -o a.out
      artifacts:
      - a.out

### Name

The name of the build, if no name is given then the default name will be in the
format of `<stage>.<n>` where `<n>` is the number of that job in the stage, for
example, `test.1`, or `build.1`.

### Stage

The name of the stage the job belongs to. If the given stage name does not exist
then the job will be ignored.

### Commands

The list of commands to run during job execution. Each command should be it's
own separate item. A command can be any string that is valid by the shell that
is interpreting it, this can vary depending on the driver being used.

### Artifacts

The list of files to collect from the build environment upon job completion.
This can use the `=>` notation to specify the name the artifact should be
collected as,

    artifacts:
    - a.out => program

## Example manifest

Below is an example manifest with all of the possible properties it could have
to demonstrate its structure,

    namespace: djinn
    driver:
      type: qemu
      image: centos/7
    env:
    - PGADDR=host=localhost port=5432 dbname=djinn user=djinn password=secret sslmode=disable
    - RDADDR=localhost:6379
    - LDFLAGS=-s -w
    objects:
    - server.crt
    - server.key
    sources:
    - https://github.com/djinn-ci/djinn.git => djinn
    stages:
    - test
    - make
    jobs:
    - stage: test
      commands:
      - cd djinn
      - go test -cover ./...
      - go test -tags "integration" ./integration
    - stage: make
      commands:
      - cd djinn
      - ./make.sh runner
      - ./make.sh server
      - ./make.sh worker
      artifacts:
      - integration/server.log
      - djinn.out
      - djinn-server.out
      - djinn-worker.out
