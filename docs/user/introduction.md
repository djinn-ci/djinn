[Prev](/user) - [Next](/user/the-basics)

# Introduction

* [What is Djinn](#what-is-djinn)
* [Authentication](#authentication)
* [Getting started](#getting-started)

## What is Djinn

Djinn is a continuous integration server. Builds that are submitted to it are
executed via drivers as specified in the build manifest. Builds can be submitted
either manually via the UI or [API](/api), or automatically from a Git
provider's webhook on commits and pull requests.

Djinn allows for the organizing of builds and build resources into namespaces.
Namespaces can have users invited to them as collaborators to work together. The
server also allows for the collection of artifacts from completed builds, and
the placement of objects into a build environment.

## Authentication

To begin using Djinn as your CI server you must first create an account. This
can either be a standalone Djinn account, or you can login via a Git provider
such as GitHub or GitLab.

## Getting started

Now that you have an account created and can login, we can get started with
submitting builds to the server. We'll start of simple by covering the basics
of Djinn, [read more...](/user/the-basics)
