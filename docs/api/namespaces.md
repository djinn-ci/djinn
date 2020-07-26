# Namespaces

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

## List namespaces for the authenticated user

This will list all of the namespaces that the currently authenticated user has
access to. This requires the explicit `namespace:read` permission.

### Request

    GET /namespaces

**Query Parameters**

| Name     | Type     | Required  | Description                                         |
|----------|----------|-----------|-----------------------------------------------------|
| `search` | `string` | N         | Get the namespaces with paths like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/namespaces


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/namespaces?search=djinn

### Response

    200 OK
    Content-Length: 5612
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/namespaces?page=1>; rel="prev",
          <https://api.djinn-ci.com/namespaces?page=3>; rel="next"
    [{
        "id": 3,
        "user_id": 1,
        "root_id": 3,
        "parent_id": null,
        "name": "djinn",
        "path": "djinn",
        "description": "",
        "visibility": "private",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/n/me/djinn",
        "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
        "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
        "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
        "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
        "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
        "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
        "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
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
            "url": "https://api.djinn-ci.com/b/me/3",
            "objects_url": "https://api.djinn-ci.com/b/me/3/objects",
            "variables_url": "https://api.djinn-ci.com/b/me/3/variables",
            "jobs_url": "https://api.djinn-ci.com/b/me/3/jobs",
            "artifacts_url": "https://api.djinn-ci.com/b/me/3/artifacts",
            "tags_url": "https://api.djinn-ci.com/b/me/3/tags",
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
    }]

>**Note:** If a namespace has a parent, then that will be present under the
`parent` field of each JSON object. The `build` field in each object will be the
last build that was submitted to the namespace if there is one, otherwise this
field will be omitted.

Nullable fields:

* `parent_id`

## Create a namespace for the authenticated user

This will create a namespace for the currently authenticated user. This requires
the explicit `namespace:write` permission.

### Request

    POST /namespaces

**Body**

| Name          | Type     | Required  | Description                               |
|---------------|----------|-----------|-------------------------------------------|
| `parent`      | `string` | N         | The name of the parent for the namespace. |
| `name`        | `string` | Y         | The name of the new namespace.            |
| `description` | `string` | N         | The description of the new namespace.     |
| `visibility`  | `string` | Y         | The visibility of the namespace.          |

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "djinn", "visibility": "private"}'\
           https://api.djinn-ci.com/namespaces

The `visibility` parameter must be one of `public`, `internal`, or `private`.

### Response

    201 Created
    Content-Length: 755
    Content-Type: application/json; charset=utf-8
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
        "url": "https://api.djinn-ci.com/n/me/djinn",
        "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
        "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
        "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
        "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
        "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
        "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
        "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }

If any of the required request paramrters are missing, or of an invalid value,
then a `400 Bad Request` response is sent back, detailing the errors for each
parameter.

    400 Bad Request
    Content-Length: 48
    Content-Type: application/json; charset=utf-8
    {"name": ["Name must be between 3 and 32 characters in length"]}

A `422 Unprocessable Entity` response will be sent back if the parent namespace
cannot be found.

    422 Unprocessable Entity
    Content-Length: 67
    Content-Type: application/json; charset=utf-8
    {"message":"Could not find parent"}

Nullable fields:

* `parent_id`

## Get an individual namespace

This will get the given namespace. This requires the explicit `namespace:read`
permission.

### Request

    GET /n/:user/:path

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn

### Response

    200 OK
    Content-Length: 755
    Content-Type: application/json; charset=utf-8
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
        "url": "https://api.djinn-ci.com/n/me/djinn",
        "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
        "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
        "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
        "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
        "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
        "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
        "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }

If the namespace has a parent then the `parent` field with the namespace's
parent.

## Update a namespace

This will update the given namespace. This requires the explicit
`namespace:write` permission.

### Request

    PATCH /n/:user/:path

**Body**

| Name          | Type     | Required  | Description                               |
|---------------|----------|-----------|-------------------------------------------|
| `user`        | `string` | Y         | The user that owns the namespace.         |
| `path`        | `string` | Y         | The path of the namespace.                |
| `description` | `string` | N         | The description of the new namespace.     |
| `visibility`  | `string` | Y         | The visibility of the namespace.          |

**Examples**

    $ curl -X PATCH \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"visibility": "internal"}'\
           https://api.djinn-ci.com/n/me/djinn

The `visibility` parameter must be one of `public`, `internal`, or `private`.

>**Note:** changing the visibility of a namespace will only take affect on a
root namespace and all of its children. You cannot change the visibility of a
child namespace independently.

### Response

    200 OK
    Content-Length: 755
    Content-Type: application/json; charset=utf-8
    {
        "id": 3,
        "user_id": 1,
        "root_id": 3,
        "parent_id": null,
        "name": "djinn",
        "path": "djinn",
        "description": "",
        "visibility": "internal",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/n/me/djinn",
        "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
        "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
        "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
        "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
        "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
        "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
        "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }

## Delete a namespace

This will delete the given namespace, and all of the given namespace's children.
This requires the explicit `namespace:delete` permission. This will set the
`namespace_id` on all of the resources and builds in the namespace to `NULL`
upon deletion.

### Request

    DELETE /n/:user/:path

**URI Parameters**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/lantern

### Response

    204 No Content

| Name     | Type     | Required  | Description                       |
|----------|----------|-----------|-----------------------------------|
| `user`   | `string` | Y         | The user that owns the namespace. |
| `path`   | `string` | Y         | The path of the namespace.        |

**Examples**

## List a namespace's builds

This will list all of the builds submitted to the given namespace. This requires
the explicit `namespace:read` permission.

### Request

    GET /n/:user/:path/-/builds

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Query Parameters**

| Name     | Type     | Required | Description                                    |
|----------|----------|----------|------------------------------------------------|
| `tag`    | `string` | N        | Get the builds with the given tag name.        |
| `search` | `string` | N        | Get the builds with tags like the given value. |
| `status` | `string` | N        | Get the builds with the given status.          |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/builds


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/builds?search=go&status=finished

### Response

    200 OK
    Content-Length: 1440
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/n/me/djinn/-/builds?page=1>; rel="prev",
          <https://api.djinn-ci.com/n/me/djinn/-/builds?page=3>; rel="next"
    [{
        "id": 3,
        "user_id": 1,
        "namespace_id": 3,
        "manifest": "namespace: djinn\ndriver:\n  image: centos/7\n  type: qemu\nenv:\n- LOCALE=en_GB.UTF-8\nobjects:\n- data => data\nstages:\n- clean\njobs:\n- stage: clean\n  commands:\n  - tr -d '0-9' data > data.cleaned\n  artifacts:\n  - data.cleaned => data.cleaned",
        "status": "queued",
        "output": null,
        "created_at": "2006-01-02T15:04:05Z",
        "started_at": null,
        "finished_at": null,
        "url": "https://api.djinn-ci.com/b/me/3",
        "objects_url": "https://api.djinn-ci.com/b/me/3/objects",
        "variables_url": "https://api.djinn-ci.com/b/me/3/variables",
        "jobs_url": "https://api.djinn-ci.com/b/me/3/jobs",
        "artifacts_url": "https://api.djinn-ci.com/b/me/3/artifacts",
        "tags_url": "https://api.djinn-ci.com/b/me/3/tags",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
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
            "url": "https://api.djinn-ci.com/n/me/djinn",
            "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
            "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
            "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
            "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
            "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
            "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
            "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            }
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
    }]

## List a namespace's namespaces

This will list all of the child namespace's for the given namespace. This
requires the explicit `namespace:read` permission.

### Request

    GET /n/:user/:path/-/namespaces

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Query Parameters**

| Name     | Type     | Required  | Description                                     |
|----------|----------|-----------|-------------------------------------------------|
| `search` | `string` | N         | Get the builds with tags like the given value.  |


**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/namespaces


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/namespaces?search=lantern

### Response

    200 OK
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/n/me/djinn/-/namespaces?page=1>; rel="prev",
          <https://api.djinn-ci.com/n/me/djinn/-/namespaces?page=3>; rel="next"
    [{
        "id": 9,
        "user_id": 1,
        "root_id": 3,
        "parent_id": 3,
        "name": "lantern",
        "path": "djinn/lantern",
        "description": "",
        "visibility": "private",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/n/me/djinn/lantern",
        "builds_url": "https://api.djinn-ci.com/n/me/djinn/lantern/-/builds",
        "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/lantern/-/namespaces",
        "images_url": "https://api.djinn-ci.com/n/me/djinn/lantern/-/images",
        "objects_url": "https://api.djinn-ci.com/n/me/djinn/lantern/-/objects",
        "keys_url": "https://api.djinn-ci.com/n/me/djinn/lantern/-/keys",
        "variables_url": "https://api.djinn-ci.com/n/me/djinn/lantern/-/variables",
        "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/lantern/-/collaborators",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        },
        "parent": {

        }
    }]

## List a namespace's images

This will list the images for the given namespace. This requires the explicit
`namespace:read` permission.

### Request

    GET /n/:user/:path/-/images

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Query Parameters**

| Name     | Type     | Required  | Description                                     |
|----------|----------|-----------|-------------------------------------------------|
| `search` | `string` | N         | Get the images with names like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/images


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/images?search=dev

### Response

    200 OK
	Content-Length: 903
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/n/me/djinn/-/images?page=1>; rel="prev",
          <https://api.djinn-ci.com/n/me/djinn/-/images?page=3>; rel="next"
    [{
        "id": 2,
        "user_id": 1,
        "namespace_id": 3,
        "name": "ubuntu",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/images/2",
        "user": {
            "id": 1,
             "email": "me@example.com",
             "username": "me",
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
            "url": "https://api.djinn-ci.com/n/me/djinn",
            "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
            "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
            "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
            "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
            "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
            "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
            "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            }
        }
    }]

## List a namespace's objects

This will list the objects for the given namespace. This requires the explicit
`namespace:read` permission.

### Request

    GET /n/:user/:path/-/objects

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Query Parameters**

| Name     | Type     | Required  | Description                                      |
|----------|----------|-----------|--------------------------------------------------|
| `search` | `string` | N         | Get the objects with names like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/objects


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/objects?search=data

### Response

    200 OK
	Content-Length: 1159
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/n/me/djinn/-/objects?page=1>; rel="prev",
          <https://api.djinn-ci.com/n/me/djinn/-/objects?page=3>; rel="next"
    [{
        "id": 1,
        "user_id": 1,
        "namespace_id": 3,
        "name": "data",
        "type": "text/plain; charset=utf-8",
        "size": 4097,
        "md5": "45ff663815a1a57ff3e24f51992238f8",
        "sha256": "2cc0ce967ed630d79f9db9e694e620f19f79afeebd0e1d2928feff773e8a7129",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/api/objects/1",
        "user": {
            "id": 1,
             "email": "me@example.com",
             "username": "me",
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
            "url": "https://api.djinn-ci.com/n/me/djinn",
            "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
            "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
            "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
            "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
            "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
            "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
            "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            }
        }
    }]

## List a namespace's variables

This will list the variables for the given namespace. This requires the explicit
`namespace:read` permission.

### Request

    GET /n/:user/:path/-/variables

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Query Parameters**

| Name     | Type     | Required  | Description                                       |
|----------|----------|-----------|---------------------------------------------------|
| `search` | `string` | N         | Get the variables with keys like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/variables


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/variables?search=EDITOR

### Response

    200 OK
	Content-Length: 1011
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/n/me/djinn/-/variables?page=1>; rel="prev",
          <https://api.djinn-ci.com/n/me/djinn/-/variables?page=3>; rel="next"
    [{
        "id": 1,
        "user_id": 1,
        "namespace_id": 3,
        "key": "EDITOR",
        "value": "ed",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/api/variables/1",
        "user": {
            "id": 1,
             "email": "me@example.com",
             "username": "me",
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
            "url": "https://api.djinn-ci.com/n/me/djinn",
            "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
            "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
            "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
            "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
            "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
            "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
            "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            }
        }
    }]

## List a namespace's keys

This will list the keys for the given namespace. This requires the explicit
`namespace:read` permission.

### Request

    GET /n/:user/:path/-/keys

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Query Parameters**

| Name     | Type     | Required  | Description                                   |
|----------|----------|-----------|-----------------------------------------------|
| `search` | `string` | N         | Get the keys with names like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/keys


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/keys?search=id_rsa

### Response

    200 OK
	Content-Length: 1011
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/n/me/djinn/-/variables?page=1>; rel="prev",
          <https://api.djinn-ci.com/n/me/djinn/-/variables?page=3>; rel="next"
    [{
        "id": 1,
        "user_id": 1,
        "namespace_id": 3,
        "name": "id_rsa",
        "config": "UserKnownHostsFile /dev/null",
        "created_at": "2006-01-02T15:04:05Z",
        "updated_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/api/keys/1",
        "user": {
            "id": 1,
             "email": "me@example.com",
             "username": "me",
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
            "url": "https://api.djinn-ci.com/n/me/djinn",
            "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
            "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
            "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
            "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
            "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
            "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
            "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            }
        }
    }]

## List a namespace's invites

This will list the invites sent for the given namespace. This requires the
explicit `namespace:read` permission. If the currently authenticated user does
not own the given namespace, then a `404 Not Found` response will be sent.

### Request

    GET /n/:user/:path/-/invites

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/invites

### Response

    200 OK
    Content-Length: 1064
    Content-Type: application/json; charset=utf-8
    [{
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
            "url": "https://api.djinn-ci.com/n/me/djinn",
            "builds_url": "https://api.djinn-ci.com/n/me/djinn/-/builds",
            "namespaces_url": "https://api.djinn-ci.com/n/me/djinn/-/namespaces",
            "images_url": "https://api.djinn-ci.com/n/me/djinn/-/images",
            "objects_url": "https://api.djinn-ci.com/n/me/djinn/-/objects",
            "variables_url": "https://api.djinn-ci.com/n/me/djinn/-/variables",
            "keys_url": "https://api.djinn-ci.com/n/me/djinn/-/keys",
            "collaborators_url": "https://api.djinn-ci.com/n/me/djinn/-/collaborators",
            "user": {
                "id": 1,
                "email": "me@example.com",
                "username": "me",
                "created_at": "2006-01-02T15:04:05Z"
            }
        },
        "inviter": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        },
        "invitee": {
            "id": 2,
            "email": "you@example.com",
            "username": "you",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }]

## List a namespace's collaborators

This will list the collaborators for the given namespace. This requires the
explicit `namespace:read` permission. If the currently authenticated user does
not own the given namespace, then a `404 Not Found` response will be sent.

### Request

    GET /n/:user/:path/-/collaborators

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/n/me/djinn/-/collaborators

### Response

    200 OK
    Content-Length: 156
    Content-Type: application/json; charset=utf-8
    [{
        "id": 2,
        "email": "you@example.com",
        "username": "you",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/n/djinn/me/-/collaborators/you",
    }]

>**Note:** The `created_at` field in the returned JSON objects will be the date
on which the collaborator was added to the namespace.
