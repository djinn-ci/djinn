[Prev](/user/variables) - [Next](/user/offline-runner)

# Webhooks

Webhooks allow you to integrate with Djinn CI based on the events that happen
within Djinn CI. When one of these events is triggered, an HTTP POST request
is sent to the webhook's URL. What happens to these events when received is up
to the server that receives it. It could be used to notify people through
various communication channels, to update an issue tracker, or to kick off
another automated process.

* [Creating a webhook](#creating-a-webhook)
* [Event payloads](#event-payloads)
  * [build_submitted](#build-submitted)
  * [build_started](#build-started)
  * [build_finished](#build-finished)
  * [build_tagged](#build-tagged)
  * [collaborator_joined](#collaborator-joined)
  * [cron](#cron)
  * [images](#images)
  * [objects](#objects)
  * [variables](#variables)
  * [ssh_keys](#ssh-keys)

## Creating a webhook

Navigate to the [namespace](/user/namespaces) you want to configure the webhook
for. From the *Webhooks* tab, you will be able to create a new webhook via the
*Create webhook* button.

## Event payloads

Detailed below are the different events, and their respective payloads that can
be sent via a webhook,

### build_submitted

This event is emitted whenever a new build is submitted to the webhook's
namespace for running.

    { }

### build_started

This is event is emitted when a build begins being run.

    { }

### build_finished

This event is emitted when a build has finished being run.

    { }

### build_tagged

This event is emitted whenever a new tag is added to a build. This will not be
emitted for tags that are added to a build at the time the build is submitted.

    { }

### collaborator_joined

This event is emitted when a user invited to a namespace accepts the invite.

    { }

### cron

This event is emitted whenever a cron job is created, updated, or deleted within
a namespace.

    { }

### images

This event is emitted whenever an image is created, or deleted within a
namespace.

    { }

### objects

This event is emitted whenever an object is created, or deleted within a
namespace.

    { }

### variables

This event is emitted whenever a variable is created, or deleted within a
namespace.

    { }

### ssh_keys

This event is emitted whenever an SSH key is created, updated, or deleted
within a namespace.

    { }
