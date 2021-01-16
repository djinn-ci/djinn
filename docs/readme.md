# Documentation

Djinn CI is a continuous integration system for automating builds of a program's
source code, its features include:

* Running builds inside of Docker containers and Linux VMs
* Cron jobs for repeatable builds
* Namespaces for organizing builds, and their resources
* Custom Linux VM build images
* Integration with GitHub and GitLab for build triggers on pushes and
pull requests
* Support for multi-repository builds
* Build artifacts - collect files from the build environment
* Build objects - place files into the build environment

**Tutorial**

Get started with Djinn CI, learn how to submit your first build, organize
resources into namespaces, and setup integration with external providers,
[read more...](/tutorial)

**User Documentation**

Learn about the build server at a high level, and how to use it to run your
builds. This will cover what a build is, how they're executed, and how you can
connect to an existing Git provider to have your builds trigger whilst you
develop, [read more...](/user)

**API Documentation**

Learn how to interact with the build server via the JSON REST API. This will
cover everything from what resources the API server exposes, to how OAuth apps
can be created for interfacing with the API, [read more...](/api)

**Admin Documentation**

Learn how to deploy and administer your own installation of the build server.
This will cover how to build the server from source, the recommended
strategies for deploying it, and how it should be administered,
[read more...](/admin)
