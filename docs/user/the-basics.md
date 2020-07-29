[Prev](/user/introduction) - [Next](/user/builds)

# The basics

This document will cover the basics of using Djinn for submitting builds,
organizing them via namespaces, and setting up webhooks to have builds trigger
on push events, and pull request events. This is a surface level document on how
to use Djinn.

* [Submitting builds](#submitting-builds)
* [Using namespaces](#using-namespaces)
* [Namespace invites](#namespace-invites)
* [Setting up webhooks](#setting-up-webhooks)
* [Going forward](#going-forward)

## Submitting builds

Builds can be submitted manually via the UI, or [API](/api), or they can be
automatically submitted via a webhook on push events and pull request events
made to a Git repository. Right now, we're going to focus on submitting our
first build manually via the UI.

We can do this by clicking the *Submit* button from the dashboard home page
once we have logged in. This will take us to a form with three fields to fill
in. The three fields are detailed below,

**Manifest** - this will be the YAML manifest of the build we want submitting,
this is a required field.

**Comment** - this is an optional field, and is typically used to describe why
the build is being submitted.

**Tags** - this is an optional field, and is expected to be a comma separated
list of values to tag the build with. Tagging a build will allow you to search
for it once it has been submitted.

Let's submit the following manifest to the server,

    driver:
        type: qemu
        image: centos/7
    sources:
    - https://github.com/andrewpillar/mdsrv.git => mdsrv
    stages:
    - build
    jobs:
    - stage: build
      commands:
      - cd mdsrv
      - ./make.sh
      artifacts:
      - mdsrv.out => mdsrv

as you can see we describe multiple things in this manifest. Let's go over
the manifest fields we have set,

* `driver.type` - this specifies the type of driver we want to use to execute
our build. For all of the supported drivers and their respective manifest
configurations see the [Drivers](/user/drivers) section of the user docs.

* `driver.image` - this specifies the image of the driver we want to use. This
will exist for some drivers such as `docker` and `qemu`, but not for others.

* `sources` - here we list the source repositories we want cloned into the
build environment. This can be any list of valid repository URL that is
recognized by Git. For more information on this see the
[Sources](/user/manifest#sources) section of the user docs.

* `stages` - the stages of the build are described here. The order in which they
are listed is the order in which they will be executed.

* `jobs` - the jobs of the build are described here. Each job much mention which
stage they belong to. The `commands` property is used to specify the commands to
execute for this job. The commands are executed in the order in which they are
specified.

* `jobs.artifacts` - the artifacts we want collected from the build environment
are listed here. The alias notation, `=>`, is used here to tell the build server
to collect the target artifact under the given alias, this is optional.

For more information on build manifests see the [Manifest](/user/manifest)
section of the user docs. This will go into more detail behind each available
property that can be set in a build manifest.

Once the build has been submitted you will be redirect to the individual build
view. From here you will be able to see the output of the build, along with
the artifacts collected, and the individual output of each job executed in the
build.

## Using namespaces

If a build is not submitted to a namespace then you will not be able to share
it with anyone else and it will be considered private. Namespaces can be used
to group related builds together. Submitting a build to a namespace is done via
the `namespace` property in the manifest, like so,

    namespace: mdsrv
    driver:
        type: qemu
        image: centos/7
    ...

you do not need to create the namespace before hand for this to work. If the
given namespace does not exist then it will be created during build submission.
By default the newly created namespace will have a visibility of `private`, this
means only you, and anyone you invite to the namespace, will be able to see it.
For more information on namespaces see the [Namespaces](/user/namespaces)
section of the user docs.

## Namespace invites

Users can be added as a collaborator to a namespace via an invite. Only the
owner of a namespace can send invites, and remove collaborators. A user can
view the invites that they have received from their *Settings* page and then
under the *Invites* tab. From here they will have the option to accept or
reject each invite they have received.

Once a user has accepted an invite and become a collaborator in a namespace
they will be able to submit builds to that namespace. This is done by prefixing
the namespace owner's username to the name of the namespace in the manifest,

    namespace: me@mdsrv

this will ensure the build will be submitted to the namespace `mdsrv` for the
user called `me`. If the user submitting the build is not a collaborator in
the given namespace, then the build submission will fail.

## Setting up webhooks

So far we've looked at manual build submission via the UI. This is fine for
testing out new build manifests, but ideally we would want this to happen
automatically as we push up changes to our code, and open pull requests. This
is achieved via webhooks that can be created once we have connected to a Git
provider.

To connect to a Git provider navigate to the *Settings* page, from here you
should see one button for each Git provider that can be connected to. Simply
click on the one you want to connect to, and login.

>**Note:** If you logged in via a Git provider then this will not be necessary.
Also, the available Git providers you can connect to may differ depending on
how the Djinn server is configured.

Once connected, navigate to the *Repos* link in the sidebar, from here you will
see the list of repositories that can be enabled for Djinn to receive webhooks
from. Simply click on the *Enable* button beside each repository you want.

In order for a webhook to successfully trigger a build, you will need to add
a build manifest to the repository. When a webhook comes in, Djinn will search
the repository for a build manifest, to do this it will look in two places.
First, it will look in the top-level of the repository for a `.djinn.yml` file,
then it will look for a `.djinn` directory, within which it will look for any
`*.yml` or `*.yaml` file to use. The server will then attempt to decode any
valid manifest from the files it finds, and submit each one to the server.

For push events the commit message will be used for the build's comment, and for
pull request events the title of the pull request will be the build's comment.

## Going forward

Now that we have a basic idea as to how builds can be submitted manually and
automatically we can start drilling down into some of the details of Djinn
and how we can make the most of it. Listed below are the pages of documentation
that is considered recommended reading for Djinn,

* [Manifest](/user/manifest) - understand what the build manifest is, how it is
structured and what you can do with a build.
* [Builds](/user/builds) - understand how builds are executed under the hood
and the order in which everything happens during execution.
* [Namespaces](/user/namespaces) - see how namespaces can be used to organize
builds and their resources for collaboration across a team.
* [Drivers](/user/drivers) - see what drivers are supported by Djinn for build
execution, and how they can be configured for you build.
* [Keys](/user/keys) - see how SSH keys can be added to Djinn to allow for the
cloning of private Git repositories within your build.
