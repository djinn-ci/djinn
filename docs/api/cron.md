[Prev](/api/builds) - [Next](/api/images)

# Cron

* [List cron jobs for the authenticated user](#list-cron-jobs-for-the-authenticated-user)
* [Create a cron job for the authenticated user](#create-a-cron-job-for-the-authenticated-user)
* [Get an individual cron job](#get-an-individual-cron-job)
* [Get a cron job's builds](#get-a-cron-jobs-builds)
* [Update a cron job](#update-a-cron-job)
* [Delete a cron job](#delete-a-cron-job)

## List cron jobs for the authenticated user

This will get the cron jobs for the currently authenticated user. This
requires the explicit `cron:read` permission.

### Request

    GET /cron

**Query Parameters**

| Name     | Type     | Required  | Description                                    |
|----------|----------|-----------|------------------------------------------------|
| `search` | `string` | N         | Get the crons with names like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/cron


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/cron?search=dail

### Response

    200 OK
    Content-Length: 1140
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/cron?page=1>; rel="prev",
          <https://api.djinn-ci.com/cron?page=3>; rel="next"
    [{
        "id": 2,
        "user_id": 1,
        "namespace_id": null,
        "name": "Nightly",
        "schedule": "daily",
        "manifest": "driver:\n  image: centos/7\n  type: qemu",
        "next_run": "2006-01-03T00:00:00Z",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/cron/2",
        "user": {
          "id": 1,
          "email": "me@example.com",
          "username": "me",
          "created_at": "2006-01-02T15:04:05Z"
        }
    }]


## Create a cron job for the authenticated user

This will create a cron job for the currently authenticated user. This requires
the explicit `cron:write` permission.

### Request

    POST /cron


**Body**

| Name        | Type     | Required | Description                                                           |
|-------------|----------|----------|-----------------------------------------------------------------------|
| `namespace` | `string` | N        | The name of the namespace to put the cron job in.                     |
| `name`      | `string` | Y        | The name of the cron job.                                             |
| `schedule`  | `string` | N        | The cron job's schedule, must be one of `daily`, `weekly`, `monthly`. |
| `manifest`  | `string` | Y        | The manifest to use for the cron job.                                 |

>**Note:** If `schedule` is not given then the schedule will be set to `daily`.

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "Daily", "manifest": "driver:\n  image: centos/7\n  type: qemu"}' \
           https://api.djinn-ci.com/cron

    200 OK
    Content-Length: 1140
    Content-Type: application/json; charset=utf-8
    {
        "id": 3,
        "user_id": 1,
        "namespace_id": null,
        "name": "Daily",
        "schedule": "daily",
        "manifest": "driver:\n  image: centos/7\n  type: qemu",
        "next_run": "2006-01-03T00:00:00Z",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/cron/3",
        "user": {
          "id": 1,
          "email": "me@example.com",
          "username": "me",
          "created_at": "2006-01-02T15:04:05Z"
        }
    }

## Get an individual cron job

This will get the given cron job. This requires the explicit `cron:read`
permission.

### Request

    GET /cron/:cron

**URI Parameters**

| Name   | Type  | Required  | Description             |
|--------|-------|-----------|-------------------------|
| `cron` | `int` | Y         | The id of the cron job. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/cron/1

### Response

    200 OK
    Content-Length: 1140
    Content-Type: application/json; charset=utf-8
    {
        "id": 2,
        "user_id": 1,
        "namespace_id": null,
        "name": "Nightly",
        "schedule": "daily",
        "manifest": "driver:\n  image: centos/7\n  type: qemu",
        "next_run": "2006-01-03T00:00:00Z",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/cron/2",
        "user": {
          "id": 1,
          "email": "me@example.com",
          "username": "me",
          "created_at": "2006-01-02T15:04:05Z"
        }
    }

## Get a cron job's builds

This will get the given cron job's builds. This requires the explicit
`cron:read` permission.

### Request

    GET /cron/:cron/builds

**URI Parameters**

| Name   | Type  | Required  | Description             |
|--------|-------|-----------|-------------------------|
| `cron` | `int` | Y         | The id of the cron job. |

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
           https://api.djinn-ci.com/cron/1/builds


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/cron/1/builds?search=go&status=finished

### Response

    200 OK
    Content-Length: 1721
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/cron/1/builds?page=1>; rel="prev",
          <https://api.djinn-ci.com/cron/1/builds?page=3>; rel="next"
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
            "type": "schedule",
            "comment": "Scheduled build, next run 2006-01-03T00:00:00Z",
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

## Update a cron job

This will update the given cron job. This requires the explicit `cron:write`
permission.

### Request

    PATCH /cron/:cron

**URI Parameters**

| Name   | Type  | Required  | Description             |
|--------|-------|-----------|-------------------------|
| `cron` | `int` | Y         | The id of the cron job. |

**Body**

>**Note:** All body parameters in this request are optional. If no body is sent
then the cron job is not updated.

| Name        | Type     | Required | Description                                                           |
|-------------|----------|----------|-----------------------------------------------------------------------|
| `name`      | `string` | N        | The name of the cron job.                                             |
| `schedule`  | `string` | N        | The cron job's schedule, must be one of `daily`, `weekly`, `monthly`. |
| `manifest`  | `string` | N        | The manifest to use for the cron job.                                 |

**Examples**

    $ curl -X PATCH \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "Daily build"}'\
           https://api.djinn-ci.com/cron/1

### Response

    200 OK
    Content-Length: 1140
    Content-Type: application/json; charset=utf-8
    {
      "id": 2,
      "user_id": 1,
      "namespace_id": null,
      "name": "Daily",
      "schedule": "daily",
      "manifest": "driver:\n  image: centos/7\n  type: qemu",
      "next_run": "2006-01-03T00:00:00Z",
      "created_at": "2006-01-02T15:04:05Z",
      "url": "https://api.djinn-ci.com/cron/1"
    }

## Delete a cron job

This will delete the given cron job. This requires the explicit `cron:delete`
permission.

### Request

    DELETE /cron/:cron

**URI Parameters**

| Name   | Type  | Required  | Description             |
|--------|-------|-----------|-------------------------|
| `cron` | `int` | Y         | The id of the cron job. |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/cron/1

### Response

    204 No Content
