[Prev](/user/variables) - [Next](/user/offline-runner)

# Webhooks

Webhooks allow you to integrate with Djinn CI based on the events that happen
within Djinn CI. When one of these events is triggered, an HTTP POST request
is sent to the webhook's URL. What happens to these events when received is up
to the server that receives it. It could be used to notify people through
various communication channels, to update an issue tracker, or to kick off
another automated process.

* [Creating a webhook](#creating-a-webhook)
* [Signing webhooks](#signing-webhooks)
* [Event payloads](#event-payloads)
  * [build.submitted](#buildsubmitted)
  * [build.started](#buildstarted)
  * [build.finished](#buildfinished)
  * [build.tagged](#buildtagged)
  * [invite.sent](#invitesent)
  * [invite.accepted](#inviteaccepted)
  * [invite.rejected](#inviterejected)
  * [namespaces](#namespaces)
  * [cron](#cron)
  * [images](#images)
  * [objects](#objects)
  * [variables](#variables)
  * [ssh_keys](#ssh-keys)

## Creating a webhook

Navigate to the [namespace](/user/namespaces) you want to configure the webhook
for. From the *Webhooks* tab, you will be able to create a new webhook via the
*Create webhook* button. Webhooks can also be created via the
[REST API](/api/namespaces#create-a-webhook-for-the-namespace).

## Signing webhooks

Secrets can be set on a webhook that is used for signing the payload of the
delivered event. Webhooks with secrets will include the signature in the
request headers,

    X-Djinn-CI-Signature sha256=6a7f769...

the secret should be used from your end to compute the hash using an HMAC
digest, then compare that with what's in the header.

## Event payloads

Detailed below are the different events, and their respective payloads that can
be sent via a webhook,

### build.submitted

This event is emitted whenever a new build is submitted to the webhook's
namespace for running.

    {
        "id": 4,
        "user_id": 3,
        "namespace_id": 4,
        "number": 1,
        "manifest": "namespace: blackmesa@wallace.breen\ndriver:\n  image: golang\n  type: docker\n  workspace: /go",
        "status": "queued",
        "output": null,
        "tags": [
            "docker",
            "golang"
        ],
        "created_at": "2021-10-06T18:14:33Z",
        "started_at": null,
        "finished_at": null,
        "url": "{{index .Vars "apihost"}}/b/wallace.breen/1",
        "objects_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/objects",
        "variables_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/variables"
        "jobs_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/jobs",
        "artifacts_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/artifacts",
        "tags_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/tags",
        "user": {
            "id": 3,
            "email": "wallace.breen@black-mesa.com",
            "username": "wallace.breen",
            "created_at": "2021-10-06T18:13:11Z"
        },
        "namespace": {
            "id": 4,
            "user_id": 3,
            "root_id": 4,
            "parent_id": null,
            "name": "blackmesa",
            "path": "blackmesa",
            "description": "Black Mesa",
            "visibility": "private",
            "created_at": "2021-10-06T18:13:11Z",
            "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
            "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
            "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
            "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
            "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
            "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
            "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
            "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
            "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
            "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-06T18:13:11Z"
            }
        },
        "trigger": {
            "type": "manual",
            "comment": "Manual build submission...",
            "data": {
                "email": "gordon.freeman@black-mesa.com",
                "username": "gordon.freeman"
            }
        }
    }

### build.started

This is event is emitted when a build begins being run, this shares the same
payload as the `build.submitted` event, only the `started_at` field will not be
null.

### build.finished

This is event is emitted when a build begins being run, this shares the same
payload as the `build.submitted` event, only the `finished_at` field will not be
null.

### build.tagged

This event is emitted whenever a new tag is added to a build. This will not be
emitted for tags that are added to a build at the time the build is submitted.

    {
        "build": {
            "artifacts_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/artifacts",
            "created_at": "2021-10-02T15:50:40Z",
            "finished_at": null,
            "id": 4,
            "jobs_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/jobs",
            "manifest": "namespace: blackmesa@wallace.breen\ndriver:\n  image: golang\n  type: docker\n  workspace: /go",
            "namespace_id": 4,
            "number": 1,
            "objects_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/objects",
            "output": null,
            "started_at": null,
            "status": "queued",
            "tags": [
                "docker",
                "golang"
            ],
            "tags_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/tags",
            "url": "{{index .Vars "apihost"}}/b/wallace.breen/1",
            "user": {
                "created_at": "2021-10-02T15:50:38Z",
                "email": "wallace.breen@black-mesa.com",
                "id": 3,
                "username": "wallace.breen"
            },
            "user_id": 3,
            "variables_url": "{{index .Vars "apihost"}}/b/wallace.breen/1/variables"
        },
        "tags": [{
            "name": "docker",
            "url": "{{index .Vars "apihost"}}/b/wallace.breen/1/tags/docker"
        }, {
            "name": "golang",
            "url": "{{index .Vars "apihost"}}/b/wallace.breen/1/tags/golang"
        }],
        "url": "{{index .Vars "apihost"}}/b/wallace.breen/1/tags",
        "user": {
            "created_at": "2021-10-02T15:50:38Z",
            "email": "gordon.freeman@black-mesa.com",
            "id":1,
            "username": "gordon.freeman"
        }
    }

### invite.sent

This event is emitted when an invite is sent to a user.

    {
        "invitee": {
            "id": 1,
            "email": "gordon.freeman@black-mesa.com",
            "username": "gordon.freeman",
            "created_at": "2021-10-02T13:32:09Z"
        },
        "inviter": {
            "id": 3,
            "email": "wallace.breen@black-mesa.com",
            "username": "wallace.breen",
            "created_at": "2021-10-02T13:32:09Z"
        },
        "namespace": {
            "id": 4,
            "user_id": 3,
            "root_id": 4,
            "parent_id": null,
            "name": "blackmesa",
            "path": "blackmesa",
            "description": "Black Mesa",
            "visibility": "private",
            "created_at": "2021-10-02T13:10:15Z",
            "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
            "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
            "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
            "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
            "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
            "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
            "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
            "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
            "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
            "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-02T13:10:14Z"
            }
        }
    }

### invite.accepted

This event is emitted when an invite is accepted by a user.

    {
        "invitee": {
            "id": 1,
            "email": "gordon.freeman@black-mesa.com",
            "username": "gordon.freeman",
            "created_at": "2021-10-02T13:32:09Z"
        },
        "namespace": {
            "id": 4,
            "user_id": 3,
            "root_id": 4,
            "parent_id": null,
            "name": "blackmesa",
            "path": "blackmesa",
            "description": "Black Mesa",
            "visibility": "private",
            "created_at": "2021-10-02T13:10:15Z",
            "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
            "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
            "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
            "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
            "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
            "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
            "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
            "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
            "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
            "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-02T13:10:14Z"
            }
        }
    }

### invite.rejected

This event is emitted when an invite is rejected by a user.

    {
        "invitee": {
            "id": 1,
            "email": "gordon.freeman@black-mesa.com",
            "username": "gordon.freeman",
            "created_at": "2021-10-02T13:32:09Z"
        },
        "namespace": {
            "id": 4,
            "user_id": 3,
            "root_id": 4,
            "parent_id": null,
            "name": "blackmesa",
            "path": "blackmesa",
            "description": "Black Mesa",
            "visibility": "private",
            "created_at": "2021-10-02T13:10:15Z",
            "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
            "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
            "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
            "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
            "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
            "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
            "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
            "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
            "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
            "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-02T13:10:14Z"
            }
        }
    }

### namespaces

This event is emitted whenever a namespace is created, updated, or deleted.
Creation events for namespaces will only be emitted for child namespaces.
The `action` field will either be `created`, `updated`, or `deleted`.

    {
        "action": "updated",
        "namespace": {
            "id": 4,
            "user_id": 3,
            "root_id": 4,
            "parent_id": null,
            "name": "blackmesa",
            "path": "blackmesa",
            "description": "Black Mesa",
            "visibility": "private",
            "created_at": "2021-10-02T13:10:15Z",
            "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
            "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
            "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
            "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
            "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
            "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
            "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
            "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
            "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
            "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-02T13:10:14Z"
            }
        }
    }

### cron

This event is emitted whenever a cron job is created, updated, or deleted within
a namespace. The `action` field will either be `created`, `updated`, or
`deleted`.

    {
       "action": "created",
       "cron": {
           "id": 4,
           "user_id": 3,
           "author_id": 3,
           "namespace_id": 4,
           "name": "nightly",
           "schedule": "daily",
           "manifest": "namespace: blackmesa@wallace.breen\ndriver:\n  image: golang\n  type: docker\n  workspace: /go",
           "created_at": "0001-01-01T00:00:00Z",
           "next_run": "2021-10-07T00:00:00Z",
           "url": "{{index .Vars "apihost"}}/cron/4",
           "author": {
               "id": 3,
               "email": "wallace.breen@black-mesa.com",
               "username": "wallace.breen",
               "created_at": "2021-10-06T18:32:33Z"
           },
           "user": {
               "id": 3,
               "email": "wallace.breen@black-mesa.com",
               "username": "wallace.breen",
               "created_at": "2021-10-06T18:32:33Z"
           },
           "namespace": {
               "id": 4,
               "user_id": 3,
               "root_id": 4,
               "parent_id": null,
               "name": "blackmesa",
               "path": "blackmesa",
               "description": "Black Mesa",
               "visibility": "private",
               "created_at": "2021-10-02T13:10:15Z",
               "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
               "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
               "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
               "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
               "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
               "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
               "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
               "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
               "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
               "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
               "user": {
                   "id": 3,
                   "email": "wallace.breen@black-mesa.com",
                   "username": "wallace.breen",
                   "created_at": "2021-10-02T13:10:14Z"
               }
           }
       }
    }

### images

This event is emitted whenever an image is created, or deleted within a
namespace. The `action` field will either be `created`, `updated`, or
`deleted`.

    {
        "action": "created",
        "image": {
            "id": 1,
            "author_id": 3,
            "user_id": 3,
            "namespace_id": 4,
            "driver": "qemu",
            "name": "resonance",
            "created_at": "2021-10-06T19:02:58Z",
            "url": "{{index .Vars "apihost"}}/images/1",
            "author": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-06T19:01:36Z"
            },
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-06T19:01:36Z"
            },
            "namespace": {
                "id": 4,
                "user_id": 3,
                "root_id": 4,
                "parent_id": null,
                "name": "blackmesa",
                "path": "blackmesa",
                "description": "Black Mesa",
                "visibility": "private",
                "created_at": "2021-10-06T19:01:36Z",
                "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
                "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
                "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
                "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
                "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
                "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
                "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
                "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
                "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
                "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
                "user": {
                    "created_at": "2021-10-06T19:01:36Z",
                    "email": "wallace.breen@black-mesa.com",
                    "id": 3,
                    "username": "wallace.breen"
                }
            }
        }
    }

### objects

This event is emitted whenever an object is created, or deleted within a
namespace. The `action` field will either be `created`, `updated`, or
`deleted`.

    {
        "action": "created",
        "object": {
            "id": 1,
            "author_id": 1,
            "user_id": 3,
            "namespace_id": 4,
            "name": "crowbar",
            "type": "image/png",
            "size": 319,
            "md5": "156dd0abff851609fb5aa6f3b0294d12",
            "sha256": "a05299b5741aad88839cd9916cd5647caa1c992247ac13dd48e230b7ca335979",
            "created_at": "2021-10-07T20:21:28Z",
            "url": "{{index .Vars "apihost"}}/objects/1",
            "author": {
                "id": 1,
                "email": "gordon.freeman@black-mesa.com",
                "username": "gordon.freeman",
                "created_at": "2021-10-07T20:20:04Z"
            },
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-07T20:20:04Z"
            },
            "namespace": {
                "id": 4,
                "user_id": 3,
                "root_id": 4,
                "parent_id": null,
                "name": "blackmesa",
                "path": "blackmesa",
                "description": "Black Mesa",
                "visibility": "private",
                "created_at": "2021-10-06T19:01:36Z",
                "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
                "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
                "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
                "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
                "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
                "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
                "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
                "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
                "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
                "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
                "user": {
                    "created_at": "2021-10-06T19:01:36Z",
                    "email": "wallace.breen@black-mesa.com",
                    "id": 3,
                    "username": "wallace.breen"
                }
            }
        }
    }

### variables

This event is emitted whenever a variable is created, or deleted within a
namespace. The `action` field will either be `created`, `updated`, or
`deleted`.

    {
        "action": "created",
        "variable": {
            "id": 1,
            "author_id": 1,
            "user_id": 3,
            "namespace_id": 4,
            "key": "PGADDR",
            "value": "host=localhost port=5432",
            "url": "{{index .Vars "apihost"}}/variables/1",
            "created_at": "2021-10-07T20:21:27Z",
            "author": {
                "id": 1,
                "email": "gordon.freeman@black-mesa.com",
                "username": "gordon.freeman",
                "created_at": "2021-10-07T20:20:04Z"
            },
            "user": {
                "id": 3,
                "email": "wallace.breen@black-mesa.com",
                "username": "wallace.breen",
                "created_at": "2021-10-07T20:20:04Z"
            },
            "namespace": {
                "id": 4,
                "user_id": 3,
                "root_id": 4,
                "parent_id": null,
                "name": "blackmesa",
                "path": "blackmesa",
                "description": "Black Mesa",
                "visibility": "private",
                "created_at": "2021-10-06T19:01:36Z",
                "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
                "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
                "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
                "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
                "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
                "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
                "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
                "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
                "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
                "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
                "user": {
                    "created_at": "2021-10-06T19:01:36Z",
                    "email": "wallace.breen@black-mesa.com",
                    "id": 3,
                    "username": "wallace.breen"
                }
            }
        }
    }

### ssh_keys

This event is emitted whenever an SSH key is created, updated, or deleted
within a namespace. The `action` field will either be `created`, `updated`, or
`deleted`.

    {
        "action": "created",
        "key": {
            "id": 1,
            "author_id": 1,
            "user_id": 3,
            "namespace_id": 4,
            "name": "id_ed25519",
            "config": "",
            "created_at": "2021-10-07T20:21:28Z",
            "updated_at": "2021-10-07T20:21:28Z",
            "url": "{{index .Vars "apihost"}}/keys/1",
            "author": {
                "id": 1,
                "email": "gordon.freeman@black-mesa.com",
                "username": "gordon.freeman",
                "created_at": "2021-10-07T20:20:04Z"
            },
            "user": {
                "created_at": "2021-10-07T20:20:04Z",
                "email": "wallace.breen@black-mesa.com",
                "id": 3,
                "username": "wallace.breen"
            },
            "namespace": {
                "id": 4,
                "user_id": 3,
                "root_id": 4,
                "parent_id": null,
                "name": "blackmesa",
                "path": "blackmesa",
                "description": "Black Mesa",
                "visibility": "private",
                "created_at": "2021-10-06T19:01:36Z",
                "url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa",
                "builds_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/builds",
                "namespaces_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/namespaces",
                "images_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/images",
                "objects_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/objects",
                "variables_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/variables",
                "keys_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/keys",
                "invites_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/invites",
                "collaborators_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/collaborators",
                "webhooks_url": "{{index .Vars "apihost"}}/n/wallace.breen/blackmesa/-/webhooks",
                "user": {
                    "created_at": "2021-10-06T19:01:36Z",
                    "email": "wallace.breen@black-mesa.com",
                    "id": 3,
                    "username": "wallace.breen"
                }
            },
        }
    }
