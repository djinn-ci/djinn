[Prev](/api/keys) - [Next](/api/oauth)

# Namespaces

* [The namespace object](#the-namespace-object)
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
