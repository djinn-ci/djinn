[Prev](/api/keys) - [Next](/api/oauth)

# Namespaces

* [The namespace object](#the-namespace-object)
* [The webhook object](#the-webhook-object)
* [List namespaces for the authenticated user](#list-namespaces-for-the-authenticated-user)
* [Create a namespace for the authenticated user](#create-a-namespace-for-the-authenticated-user)
* [Get an individual namespace](#get-an-individual-namespace)
* [Update a namespace](#update-a-namespace)
* [Delete a namespace](#delete-a-namespace)
* [List a namespace's builds](#list-a-namespaces-builds)
* [List a namespace's namespaces](#list-a-namespaces-namespaces)
* [List a namespace's images](#list-a-namespaces-images)
* [List a namespace's objects](#list-a-namespaces-objects)
* [List a namespace's variables](#list-a-namespaces-variables)
* [List a namespace's keys](#list-a-namespaces-keys)
* [List a namespace's invites](#list-a-namespaces-invites)
* [List a namespace's collaborators](#list-a-namespaces-collaborators)
* [List a namespace's webhooks](#list-a-namespaces-webhooks)
* [Create a webhook for the namespace](#create-a-webhook-for-the-namespace)
* [Update a webhook in the namespace](#update-a-webhook-in-the-namespace)
* [Delete a webhook from the namespace](#delete-a-webhook-in-from-namespace)

## The namespace object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the namespace.

---

**`user_id`** `int` - ID of the user who owns the namespace.

---

**`root_id`** `int` - ID of the top-level namespace for the current namespace.
This will match the `id` attribute for the root namespace.

---

**`parent_id`** `int` `nullable` - ID of the namespace's parent, if any.

---

**`name`** `string` - The name of the current namespace.

---

**`path`** `string` - The full path of the namespace. This will include the
parent namespace names.

---

**`description`** `string` - The description of the current namespace.

---

**`visibility`** `enum` - The visibility level of the namespace, will be one
of,  `private`, `internal`, `public`.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the
namespace was created.

---

**`url`** `string` - The API URL to the namespace object itself.

---

**`builds_url`** `string` - The API URL to the namespace's builds.

---

**`namespaces_url`** `string` - The API URL to the namespace's children.

---

**`images_url`** `string` - The API URL to the namespace's images.

---

**`objects_url`** `string` - The API URL to the namespace's objects.

---

**`variables_url`** `string` - The API URL to the namespace's variables.

---

**`keys_url`** `string` - The API URL to the namespace's keys.

---

**`collaborators_url`** `string` - The API URL to the namespace's collaborators.

---

**`webhooks_url`** `string` - The API URL to the namespace's webhooks.

---

**`user`** `object` `nullable` - The [user](/api/user#the-user-object) of the
namespace, if any.

---

**`parent`** `object` `nullable` - The
[parent](/api/namespaces#the-namespace-object) of the namespace, if any.

---

**`build`** `object` `nullable` - The [build](/api/builds#the-build-object)
that was most recently submitted to the namespace.

</div>
<div class="api-example">

**Object**

    {
        "id": 3,
        "user_id": 1,
        "root_id": 3,
        "parent_id": null,
        "name": "djinn",
        "path": "djinn",
        "description": "",
        "visibility": "private",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "{{index .Vars "apihost"}}/n/me/djinn",
        "builds_url": "{{index .Vars "apihost"}}/n/me/djinn/-/builds",
        "namespaces_url": "{{index .Vars "apihost"}}/n/me/djinn/-/namespaces",
        "images_url": "{{index .Vars "apihost"}}/n/me/djinn/-/images",
        "objects_url": "{{index .Vars "apihost"}}/n/me/djinn/-/objects",
        "variables_url": "{{index .Vars "apihost"}}/n/me/djinn/-/variables",
        "keys_url": "{{index .Vars "apihost"}}/n/me/djinn/-/keys",
        "collaborators_url": "{{index .Vars "apihost"}}/n/me/djinn/-/collaborators",
        "webhooks_url": "{{index .Vars "apihost"}}/n/me/djinn/-/webhooks",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        },
        "build": {
            "id": 3,
            "user_id": 1,
            "namespace_id": 3,
            "manifest": "namespace: djinn\ndriver:\n  image: centos/7\n  type: qemu\nenv:\n- LOCALE=en_GB.UTF-8\nobjects:\n- data => data\nstages:\n- clean\njobs:\n- stage: clean\n  commands:\n  - tr -d '0-9' data > data.cleaned\n  artifacts:\n  - data.cleaned => data.cleaned",
            "status": "queued",
            "output": null,
            "created_at": "2006-01-02T15:04:05Z",
            "started_at": null,
            "finished_at": null,
            "url": "{{index .Vars "apihost"}}/b/me/3",
            "objects_url": "{{index .Vars "apihost"}}/b/me/3/objects",
            "variables_url": "{{index .Vars "apihost"}}/b/me/3/variables",
            "jobs_url": "{{index .Vars "apihost"}}/b/me/3/jobs",
            "artifacts_url": "{{index .Vars "apihost"}}/b/me/3/artifacts",
            "tags_url": "{{index .Vars "apihost"}}/b/me/3/tags",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            },
            "trigger": {
                "type": "manual",
                "comment": "",
                "data": {
                    "email": "me@example.com",
                    "username": "me"
                }
            },
            "tags": [
                "anon",
                "golang"
            ]
        }
    }

</div>
</div>

## The webhook object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the webhook.

---

**`author_id`** `int` - ID of the user who created the webhook.

---

**`user_id`** `int` - ID of the user the webhook belongs to.

---

**`namespace_id`** `int` - ID of the namespace the webhook belongs to.

---

**`payload_url`** `string` - URL to send the event payload to.

---

**`ssl`** `bool` - Whether or not the event will be sent over TLS.

---

**`active`** `bool` - Whether or not the webhook is active.

---

**`events`** `string[]` - The events the webhook will activate on. See the
[Event payloads](/user/webhooks#event-payloads) section for details on the
different webhook events.

---

**`namespace`** `object` - The [namespace](/api/namespaces#the-namespace-object)
object.

---

**`last_response`** `object` - The last response received from the webhook, if
any.

---

**`last_response.code`** `int` - The HTTP status code of the delivery

---

**`last_response.duration`** `int` - The duration of the request delivered to
the URL in nanoseconds.

---

**`last_response.error`** `string` - The error that occurred if the event failed
to be delivered. This will be `null` if not error occurred.

---

**`last_response.created_at`** `timestamp` - The RFC3339 formatted string at
which the delivery was made.

</div>

<div class="api-example">

**Object**

    {
        "id": 1,
        "user_id": 1,
        "author_id": 1,
        "namespace_id": 1,
        "payload_url": "https://api.example.com/hook/djinn-ci",
        "ssl": true,
        "active": true,
        "events": [
            "build_submitted",
            "build_finished",
            "build_tagged"
        ],
        "last_response": {
            "code": 204,
            "duration": 1024,
            "created_at": "2006-01-02T15:04:05Z"
        },
        "namespace": {
            "id": 3,
            "user_id": 1,
            "root_id": 3,
            "parent_id": null,
            "name": "djinn",
            "path": "djinn",
            "description": "",
            "visibility": "private",
            "created_at": "2006-01-02T15:04:05Z",
            "url": "{{index .Vars "apihost"}}/n/me/djinn",
            "builds_url": "{{index .Vars "apihost"}}/n/me/djinn/-/builds",
            "namespaces_url": "{{index .Vars "apihost"}}/n/me/djinn/-/namespaces",
            "images_url": "{{index .Vars "apihost"}}/n/me/djinn/-/images",
            "objects_url": "{{index .Vars "apihost"}}/n/me/djinn/-/objects",
            "variables_url": "{{index .Vars "apihost"}}/n/me/djinn/-/variables",
            "keys_url": "{{index .Vars "apihost"}}/n/me/djinn/-/keys",
            "collaborators_url": "{{index .Vars "apihost"}}/n/me/djinn/-/collaborators",
            "webhooks_url": "{{index .Vars "apihost"}}/n/me/djinn/-/webhooks",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            }
        }
    }

</div>
</div>

## List namespaces for the authenticated user

<div class="api-section">
<div class="api-doc">

This will list all of the namespaces that the currently authenticated user has
access to. This requires the explicit `namespace:read` permission.

**Parameters**

---

**`search`** `string` *optional* - Get the namespaces with paths like the given value.

**Returns**

Returns a list of [namespaces](/api/namespaces#the-namespace-object). The list
will be paginated to 25 namespaces per page and will be ordered by the most
recently created namespace first. If the namespaces were paginated, then the
pagination information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/namespaces?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/namespaces?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /namespaces

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/namespaces


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/namespaces?search=djinn

</div>
</div>

## Create a namespace for the authenticated user

<div class="api-section">
<div class="api-doc">

This will create a namespace for the currently authenticated user. This requires
the explicit `namespace:write` permission.

**Parameters**

---

**`parent`** `string` *optional* - The name of the parent for the namespace.

---

**`name`** `string` - The name of the new namespace.

---

**`description`** `string` *optional* - The description of the new namespace.

---

**`visibility`** `enum` - The visibility of the namespace, must be one of
`private`, `internal`, or `public`.

**Returns**

Returns the [namespace](/api/namespaces#the-namespace-object). It returns an
[error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

**Request**

    POST /namespaces

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "djinn", "visibility": "private"}'\
           {{index .Vars "apihost"}}/namespaces

</div>
</div>

## Get an individual namespace

<div class="api-section">
<div class="api-doc">

This will get the given namespace. This requires the explicit `namespace:read`
permission.

**Returns**

Returns the [namespace](/api/namespaces#the-namespace-object).

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn

</div>
</div>

## Update a namespace

<div class="api-section">
<div class="api-doc">

This will update the given namespace. This requires the explicit
`namespace:write` permission.

**Parameters**

---

**`description`** `string` *optional* - The description of new namespace.

---

**`visibility`** `enum` - The visibility of the namespace, must be one of
`private`, `internal`, or `public`.

>**Note:** changing the visibility of a namespace will only take affect on a
root namespace and all of its children. You cannot change the visibility of a
child namespace independently.

**Returns**

Returns the [namespace](/api/namespaces#the-namespace-object).

</div>
<div class="api-example">

**Request**

    PATCH /n/:user/:path

**Examples**

    $ curl -X PATCH \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"visibility": "internal"}'\
           {{index .Vars "apihost"}}/n/me/djinn

</div>
</div>

## Delete a namespace

<div class="api-section">
<div class="api-doc">

This will delete the given namespace, and all of the given namespace's children.
This requires the explicit `namespace:delete` permission. This will set the
`namespace_id` on all of the resources and builds in the namespace to `NULL`
upon deletion.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /n/:user/:path

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/lantern

</div>
</div>

## List a namespace's builds

<div class="api-section">
<div class="api-doc">

This will list all of the builds submitted to the given namespace. This requires
the explicit `namespace:read` permission.

**Parameters**

---

**`tag`** `string` *optional* - Get the builds with the given tag name.

---

**`search`** `string` *optional* - Get the builds with tags like the given value.

---

**`status`** `string` *optional* - Get the builds with the given status.

**Returns**

Returns a list of [builds](/api/builds#the-build-object) for the given
namespace. The list will be paginated to 25 builds per page and will be ordered
by the most recently submitted builds first. If the builds were paginated, then
the pagination information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/n/me/djinn/-/builds?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/n/me/djinn/-/builds?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/builds

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/builds


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/builds?search=go&status=finished

</div>
</div>

## List a namespace's namespaces

<div class="api-section">
<div class="api-doc">

This will list all of the child namespace's for the given namespace. This
requires the explicit `namespace:read` permission.

**Parameters**

---

**`search`** `string` *optional* - Get the builds with tags like the given value.

**Returns**

Returns a list of [namespaces](/api/namespaces#the-namespace-object). The list
will be paginated to 25 namespaces per page and will be ordered by the most
recently created namespace first. If the namespaces were paginated, then the
pagination information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/n/me/djinn/-/namespaces?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/n/me/djinn/-/namespaces?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/namespaces


**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/namespaces


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/namespaces?search=lantern

</div>
</div>

## List a namespace's images

<div class="api-section">
<div class="api-doc">

This will list the images for the given namespace. This requires the explicit
`namespace:read` permission.

**Parameters**

**`search`** `string` `optional` - Get the images with names like the given value.

**Returns**

Returns a list of [images](/api/images#the-image-object) for the namespace. The
list will be paginated to 25 images per page and will be ordered by the most
recently created image first. If the images were paginated, then the pagination
information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/n/me/djinn/-/images?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/n/me/djinn/-/images?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/images

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/images


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/images?search=dev

</div>
</div>

## List a namespace's objects

<div class="api-section">
<div class="api-doc">

This will list the objects for the given namespace. This requires the explicit
`namespace:read` permission.

**Parameters**

---

**`search`** `string` *optional* - Get the objects with names like the given value.

**Returns**

Returns a list of [objects](/api/object#the-object-object) for the
namespace. The list will be paginated to 25 objects per page and will be ordered
by the most recently created object first. If the objects were paginated, then
the pagination information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/n/me/djinn/-/objects?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/n/me/djinn/-/objects?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/objects

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/objects


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/objects?search=data

</div>
</div>

## List a namespace's variables

<div class="api-section">
<div class="api-doc">

This will list the variables for the given namespace. This requires the explicit
`namespace:read` permission.

**Parameters**

---

**`search`** `string` *optional* - Get the variables with keys like the given value.

**Returns**

Returns a list of [variables](/api/variables#the-variable-object) for the
namespace. The list will be paginated to 25 variables per page and will be
ordered by the most recently created variable first. If the objects were
paginated, then the pagination information will be in the response header
`Link` like so,

    Link: <{{index .Vars "apihost"}}/n/me/djinn/-/variables?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/n/me/djinn/-/variables?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/variables

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/variables


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/variables?search=EDITOR

</div>
</div>

## List a namespace's keys

<div class="api-section">
<div class="api-doc">

This will list the keys for the given namespace. This requires the explicit
`namespace:read` permission.

**Parameters**

---

**`search`** `string` *optional* - Get the keys with names like the given value.

**Returns**

Returns a list of [keys](/api/keys#the-key-object) for the namespace. The list
will be paginated to 25 keys per page and will be ordered by the most recently
created key first. If the keys were paginated, then the pagination information
will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/n/me/djinn/-/keys?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/n/me/djinn/-/keys?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/keys

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/keys


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/keys?search=id_rsa

</div>
</div>

## List a namespace's invites

<div class="api-section">
<div class="api-doc">

This will list the invites sent for the given namespace. This requires the
explicit `namespace:read` permission. If the currently authenticated user does
not own the given namespace, then a `404 Not Found` response will be sent.

**Returns**

Returns a list of [invites](/api/invites#the-invite-object).

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/invites

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/invites

</div>
</div>

## List a namespace's collaborators

<div class="api-section">
<div class="api-doc">

This will list the collaborators for the given namespace. This requires the
explicit `namespace:read` permission. If the currently authenticated user does
not own the given namespace, then a `404 Not Found` response will be sent.

**Returns**

Returns a list of [collaborators](/api/user#the-user-object) in the namespace.

>**Note:** The `created_at` field in the returned JSON objects will be the date
on which the collaborator was added to the namespace.

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/collaborators

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/collaborators

</div>
</div>

## List a namespace's webhooks

<div class="api-section">
<div class="api-doc">

This will list the webhooks for the given namespace. This requires the explicit
`webhook:read` permission. If the currently authenticated user does not own the
given namespace, then a `404 Not Found` response will be sent.

**Returns**

Returns a list of [webhooks](/api/namespaces#the-webhook-object) in the
namespace.

</div>
<div class="api-example">

**Request**

    GET /n/:user/:path/-/webhooks

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/n/me/djinn/-/webhooks

</div>
</div>

## Create a webhook for the namespace

<div class="api-section">
<div class="api-doc">

This will create a webhook for the currently authenticated user in the given
namespace. This requires the explicit `webhook:write` permission.

**Parameters**

---

**`payload_url`** `string` - The URL to send the event payload to.

---

**`secret`** `string` - *optional* The secret to sign the event payload with.

---

**`ssl`** `bool` - *optional* Whether or not to use TLS when sending the request.

---

**`active`** `bool` - *optional* Whether or not the webhook should be active.

---

**`events`** `string[]` - *optional* The events for the webhook to activate on.
If no events are given, then the webhook will activate on all events.

**Returns**

Returns the [webhook](/api/namespaces#the-webhook-object). It returns an
[error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

</div>
</div>

## Update a webhook in the namespace

<div class="api-section">
<div class="api-doc">

</div>
<div class="api-example">

</div>
</div>

## Delete a webhook from the namespace

<div class="api-section">
<div class="api-doc">

</div>
<div class="api-example">

</div>
</div>
