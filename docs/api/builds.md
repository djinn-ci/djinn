[Prev](/api) - [Next](/api/images)

# Builds

* [List builds for the authenticated user](#list-builds-for-the-authenticated-user)
* [Submit a build for the authenticated user](#submit-a-build-for-the-authenticated-user)
* [Get a build](#get-a-build)
* [Get a build's objects](#get-a-builds-objects)
* [Get a build's variables](#get-a-builds-variables)
* [Get a build's jobs](#get-a-builds-jobs)
* [Get an individual build job](#get-an-individual-build-job)
* [Get a build's artifacts](#get-a-builds-artifacts)
* [Get an individual build artifact](#get-an-individual-build-artifact)
* [Get a build's tags](#get-a-builds-tags)
* [Get an individual tag from a build](#get-an-individual-tag-from-a-build)
* [Add tags to a build](#add-tags-to-a-build)
* [Remove tags from a build](#remove-tags-from-a-build)
* [Kill a build](#kill-a-build)

## List builds for the authenticated user

This will list all of the builds that the currently authenticated user has
access to. This requires the explicit `build:read` permission for the user.

### Request

    GET /builds

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
           https://api.djinn-ci.com/builds


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/builds?search=go&status=finished

### Response

    200 OK
    Content-Length: 1721
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/builds?page=1>; rel="prev",
          <https://api.djinn-ci.com/builds?page=3>; rel="next"
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

Nullable fields:

* `namespace_id`
* `output`
* `started_at`
* `finished_at`

## Submit a build for the authenticated user

This will submit a new build to the server for the currently authenticated user.
This requires the explicit `build:write` permission.

### Request

    POST /builds

**Body**

| Name       | Type       | Required | Description                               |
|------------|------------|----------|-------------------------------------------|
| `manifest` | `string`   | Y        | The YAML formatted build manifest.        |
| `tags`     | `string[]` | N        | An array of the tags to add to the build. |

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"manifest":"namespace: djinn\ndriver:\n  image: centos/7\n  type: qemu\nenv:\n- LOCALE=en_GB.UTF-8\nobjects:\n- data => data\nstages:\n- clean\njobs:\n- stage: clean\n  commands:\n  - tr -d '0-9' data > data.cleaned\n  artifacts:\n  - data.cleaned => data.cleaned"}' \
           https://api.djinn-ci.com/builds


    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"manifest":"namespace: djinn\ndriver:\n  image: centos/7\n  type: qemu\nenv:\n- LOCALE=en_GB.UTF-8\nobjects:\n- data => data\nstages:\n- clean\njobs:\n- stage: clean\n  commands:\n  - tr -d '0-9' data > data.cleaned\n  artifacts:\n  - data.cleaned => data.cleaned", "tags":["tag1"]}' \
           https://api.djinn-ci.com/builds

### Response

    201 Created
    Content-Length: 1719
    Content-Type: application/json; charset=utf-8
    {
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
    }

If any of the required request parameters are missing, then a `400 Bad Request`
response is sent back, detailing the errors for each missing parameter.

    400 Bad Request
    Content-Length: 48
    Content-Type: application/json; charset=utf-8
    {"manifest": ["Build manifest can't be blank"]}

## Get a build

This will get the build by the given `:user`, with the given `:id`. This
requires the explicit `build:read` permission.

### Request

    GET /b/:user/:id

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3

### Response

    200 OK
    Content-Length: 1719
    Content-Type: application/json; charset=utf-8
    {
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
    }

## Get a build's objects

This will return a list of all the objects that have been placed, or will be
placed on the build. This requires the explicit `build:read` permission.

### Request

    GET /b/:user/:id/objects

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/objects

### Response

    200 OK
    Content-Length: 273
    Content-Type: application/json; charset=utf-8
    [{
        "id": 1,
        "build_id": 3,
        "source": "data",
        "name": "data",
        "type": "text/plain; charset=utf-8",
        "md5": "45ff663815a1a57ff3e24f51992238f8",
        "sha256": "2cc0ce967ed630d79f9db9e694e620f19f79afeebd0e1d2928feff773e8a7129",
        "placed": false,
        "object_url": "https://api.djinn-ci.com/objects/1",
    }]

>**Note:** The nullable fields detailed below will be null if the original
object has been deleted, and there will be no `object_url` present.

Nullable fields:

* `type`
* `md5`
* `sha256`

## Get a build's variables

This will return a list of all the variables that have been set for the build,
either set from the build manifest itself, or set via the variables resource.
This requires the explicit `build:read` permission.

### Request

    GET /b/:user/:id/variables

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/variables

### Response

    200 OK
    Content-Length: 167
    Content-Type: application/json; charset=utf-8
    [{
        "id", 2,
        "build_id": 3,
        "key": "EDITOR",
        "value": "ed",
        "variable_url": "https://api./djinn-ci.com/variables/1"
    },{
        "id", 3,
        "build_id": 3,
        "key": "LOCALE",
        "value": "ed"
    }]

>**Note:** If a build variable was created via the build manifest, then there
will be no `variable_url` present in the returned JSON object for that variable.
This will also be the case if the source variable was deleted.

## Get a build's jobs

This will return a list of all the jobs that are part of the build. This
requires the explicit `build:read` permission.

### Request

    GET /b/:user/:id/jobs

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/jobs

### Response

    200 OK
    Content-Length: 495
    Content-Type: application/json; charset=utf-8
    [{
        "id": 4,
        "build_id": 3,
        "name": "create driver",
        "commands": "",
        "status": "queued",
        "output": null,
        "created_at": "2006-01-02T15:04:05Z",
        "started_at": nil,
        "finished_at": nil,
        "url": "https://api.djinn-ci.com/b/me/3/jobs/4"
    },{
        "id": 5,
        "build_id": 3,
        "name": "clean.1,
        "commands": "tr -d '0-9' data > data.cleaned",
        "status": "queued",
        "output": null,
        "created_at": "2006-01-02T15:04:05Z",
        "started_at": nil,
        "finished_at": nil,
        "url": "https://api.djinn-ci.com/b/me/3/jobs/5"
    }]

>**Note:** The job `create driver` is a pseudo-job created by the build server
to capture the output of the driver during its creation for build execution.

Nullable fields:

* `output`
* `started_at`
* `finished_at`

## Get an individual build job

This will return an individual job for the given build. Ths requires the
explicit `build:read` permission.

### Request

    GET /b/:user/:id/jobs/:job_id

**URI Parameters**

| Name     | Type     | Required | Description                                |
|----------|----------|----------|--------------------------------------------|
| `user`   | `string` | Y        | The name of the user the build belongs to. |
| `id`     | `int`    | Y        | The id of the build to get.                |
| `job_id` | `int`    | Y        | The id of the job to get.                  |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/jobs/5

### Response

    200 OK
    Content-Length: 1746
    Content-Type: application/json; charset=utf-8
    {
        "id": 5,
        "build_id": 3,
        "stage": "clean",
        "name": "clean.1,
        "commands": "tr -d '0-9' data > data.cleaned",
        "status": "queued",
        "output": null,
        "created_at": "2006-01-02T15:04:05Z",
        "started_at": nil,
        "finished_at": nil,
        "url": "https://api.djinn-ci.com/b/me/3/jobs/5"
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
        }
    }

Nullable fields:

* `output`
* `started_at`
* `finished_at`

## Get a build's artifacts

This will list the artifacts that have been collected from the given build. This
requires the explicit `build:read` permission.

### Request

    GET /b/:user/:id/artifacts

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/artifacts

### Response

    200 OK
    Content-Length: 208
    Content-Type: application/json; charset=utf-8
    [{
        "id": 1,
        "build_id": 3,
        "job_id": 5,
        "source": "data.cleaned",
        "name": "data.cleaned",
        "size": null,
        "md5": null,
        "sha256": null,
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/b/me/3/artifacts/1"
    }]

>**Note:** The nullable fields will be set accordingly once the artifact has
been collected from the build.

Nullable fields:

* `size`
* `md5`
* `sha256`

## Get an individual build artifact

This will get an individual artifact for the given build. This requires the
explicit `build:read` permission.

### Request

    GET /b/:user/:id/artifacts/:artifact_id

**URI Parameters**

| Name          | Type     | Required | Description                                |
|---------------|----------|----------|--------------------------------------------|
| `user`        | `string` | Y        | The name of the user the build belongs to. |
| `id`          | `int`    | Y        | The id of the build to get.                |
| `artifact_id` | `int`    | Y        | The id of the artifact to get.             |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/artifacts/1

### Response

    200 OK
    Content-Length: 1058
    Content-Type: application/json; charset=utf-8
    {
        "id": 1,
        "build_id": 3,
        "job_id": 5,
        "source": "data.cleaned",
        "name": "data.cleaned",
        "size": null,
        "md5": null,
        "sha256": null,
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/b/me/3/artifacts/1",
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
        }
    }

## Get a build's tags

This will list the tags set on the given build. This requires the explicit
`build:read` permission.

### Request

    GET /b/:user/:id/tags

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/tags

### Response

    200 OK
    Content-Length: 465
    Content-Type: application/json; charset=utf-8
    [{
        "id": 3,
        "user_id": 1,
        "build_id": 3,
        "name": "centos/7",
        "created_at": "2006-01-02T15:04:05Z"
        "url": "https://api.djinn-ci.com/b/me/3/tags/3",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    },{
        "id": 4,
        "user_id": 1,
        "build_id": 3,
        "name": "another-tag",
        "created_at": "2006-01-02T15:04:05Z"
        "url": "https://api.djinn-ci.com/b/me/3/tags/4",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }]

## Get an individual tag from a build

This will get the given tag from the given build. This requires the explicit
`build:read` permission.

### Request

    GET /b/:user/:id/tags/:tag_id

**URI Parameters**

| Name     | Type     | Required | Description                                |
|----------|----------|----------|--------------------------------------------|
| `user`   | `string` | Y        | The name of the user the build belongs to. |
| `id`     | `int`    | Y        | The id of the build to get.                |
| `tag_id` | `int`    | Y        | The id of the tag to delete.               |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/tags/4

### Response

    200 OK
    Content-Length: 989
    Content-Type: application/json; charset=utf-8
    {
        "id": 4,
        "user_id": 1,
        "build_id": 3,
        "name": "another-tag",
        "created_at": "2006-01-02T15:04:05Z"
        "url": "https://api.djinn-ci.com/b/me/3/tags/4",
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
        }
    }

## Add tags to a build

This will add the given tags to the given build. This requires the explicit
`build:write` permission.

### Request

    POST /b/:user/:id/tags

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Body**

This request expects a JSON array of string values to be submitted to the
endpoint as the request body, for example,

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '["tag1", "tag2", "tag3"]' \
           https://api.djinn-ci.com/b/me/3/tags

### Response

    201 Created
    Content-Length: 401
    Content-Type: application/json; charset=utf-8
    [{
        "id": 5,
        "user_id": 1,
        "build_id": 3,
        "name": "tag1",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/b/me/3/tags/5"
    },{
        "id": 6,
        "user_id": 1,
        "build_id": 3,
        "name": "tag2",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/b/me/3/tags/6"
    },{
        "id": 7,
        "user_id": 1,
        "build_id": 3,
        "name": "tag3",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/b/me/3/tags/7"
    }]

## Remove tags from a build

This will remove the given tag from a given build. This requires the explicit
`build:delete` permission.

### Request

    DELETE /b/:user/:id/tags/:tag_id

**URI Parameters**

| Name     | Type     | Required | Description                                |
|----------|----------|----------|--------------------------------------------|
| `user`   | `string` | Y        | The name of the user the build belongs to. |
| `id`     | `int`    | Y        | The id of the build to get.                |
| `tag_id` | `int`    | Y        | The id of the tag to delete.               |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3/tags/5

### Response

    204 No Content

## Kill a build

This will kill a build that is running. This requires the explicit
`build:delete` permission.

### Request

    DELETE /b/:user/:id

**URI Parameters**

| Name   | Type     | Required | Description                                |
|--------|----------|----------|--------------------------------------------|
| `user` | `string` | Y        | The name of the user the build belongs to. |
| `id`   | `int`    | Y        | The id of the build to get.                |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/b/me/3

### Response

    204 No Content
