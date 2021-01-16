[Prev](/api/builds) - [Next](/api/images)

# Cron

* [The cron job object](#the-cron-job-object)
* [List cron jobs for the authenticated user](#list-cron-jobs-for-the-authenticated-user)
* [Create a cron job for the authenticated user](#create-a-cron-job-for-the-authenticated-user)
* [Get an individual cron job](#get-an-individual-cron-job)
* [Get a cron job's builds](#get-a-cron-jobs-builds)
* [Update a cron job](#update-a-cron-job)
* [Delete a cron job](#delete-a-cron-job)

## The cron job object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the cron job.

---

**`author_id`** `int` - ID of the user who created the cron job.

---

**`user_id`** `int` - ID of the user the cron job belongs to.

---

**`namespace_id`** `int` `nullable` - ID of the namespace the cron job belongs
to if any.

---

**`name`** `string` - The name of the cron job.

---

**`schedule`** `enum` - The schedule of the cron job, one of `daily`, `weekly`,
`monthly`.

---

**`manifest`** `string` - The build manifest the cron should submit on its
interval.

---

**`next_run`** `timestamp` - The RFC3339 formatted string at which the cron job
will next run.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the cron
job was created.

---

**`url`** `string` - The API URL to the cron job object itself.

---

**`author`** `object` `nullable` - The [user](/api/user#the-user-object) who
authored the cron job, if any.

---

**`user`** `object` `nullable` - The [user](/api/user#the-user-object) of the
cron job, if any.

---

**`namespace`** `object` `nullable` - The
[namespace](/api/namespaces#the-namespace-object) the cron job belongs to, if
any.

</div>
<div class="api-example">

**Object**

    {
        "id": 2,
        "author_id": 1,
        "user_id": 1,
        "namespace_id": null,
        "name": "Nightly",
        "schedule": "daily",
        "manifest": "driver:\n  image: centos/7\n  type: qemu",
        "next_run": "2006-01-03T00:00:00Z",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "{{index .Vars "apihost"}}/cron/2",
        "user": {
          "id": 1,
          "email": "me@example.com",
          "username": "me",
          "created_at": "2006-01-02T15:04:05Z"
        }
    }

</div>
</div>

## List cron jobs for the authenticated user

<div class="api-section">
<div class="api-doc">

This will get the cron jobs for the currently authenticated user. This
requires the explicit `cron:read` permission.

**Parameters**

---

**`search`** `string` - Get the cron jobs with names like the given value.

**Returns**

Returns a list of [cron job objects](/api/cron#the-cron-job-object). The list
will be paginated to 25 cron jobs per page and will be ordered by the most
recently created cron job first. If the cron jobs were paginated, then the
pagination information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/cron?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/cron?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /cron

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/cron


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/cron?search=dail

</div>
</div>

## Create a cron job for the authenticated user

<div class="api-section">
<div class="api-doc">

This will create a cron job for the currently authenticated user. This requires
the explicit `cron:write` permission.

**Parameters**

---

**`namespace`** `string` - The name of the namespace to put the cron job in.

---

**`name`** `string` - The name of the cron job.

---

**`schedule`** `enum` *optional* - The cron job's schedule, must be one of
`daily`, `weekly`, `monthly`. Defaults to `daily` if not given.

---

**`manifest`** `string` - The manifest to use for the cron job.

**Returns**

Returns the [cron job](/api/cron#the-cron-job-object). It returns an
[error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

**Request**

    POST /cron

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "Daily", "manifest": "driver:\n  image: centos/7\n  type: qemu"}' \
           {{index .Vars "apihost"}}/cron

</div>
</div>

## Get an individual cron job

<div class="api-section">
<div class="api-doc">

This will get the given cron job. This requires the explicit `cron:read`
permission.

**Returns**

Returns the [cron job](/api/cron#the-cron-job-object).

</div>
<div class="api-example">

**Request**

    GET /cron/:cron

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/cron/1

</div>
</div>

## Get a cron job's builds

<div class="api-section">
<div class="api-doc">

This will get the given cron job's builds. This requires the explicit
`cron:read` permission.

**Parameters**

---

**`tag`** `string` *optional* - Get the builds with the given tag name.

---

**`search`** `string` *optional* - Get the builds with tags like the given value.

---

**`status`** `string` *optional* - Get the builds with the given status.

**Returns**

Returns a list of [builds](/api/builds#the-build-object) that were submitted
from the given cron. The list will be paginated to 25 builds per page and will
be ordered by the most recently submitted builds first. If the builds were
paginated, then the pagination information will be in the response header
`Link` like so,

    Link: <{{index .Vars "apihost"}}/builds?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/builds?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /cron/:cron/builds

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/cron/1/builds


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/cron/1/builds?search=go&status=finished

</div>
</div>

## Update a cron job

<div class="api-section">
<div class="api-doc">

This will update the given cron job. This requires the explicit `cron:write`
permission.

**Parameters**

---

**`name`** `string` *optional* - The name of the cron job.

---

**`schedule`** `enum` *optional* - The cron job's schedule, must be one of
`daily`, `weekly`, `monthly`. Defaults to `daily` if not given.

---

**`manifest`** `string` *optional* - The manifest to use for the cron job.

>**Note**: If no parameters are sent in the request body then nothing happens
to the cron job.

**Returns**

Returns the updated [cron job](/api/cron#the-cron-job-object).

</div>
<div class="api-example">

**Request**

    PATCH /cron/:cron

**Examples**

    $ curl -X PATCH \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "Daily build"}'\
           {{index .Vars "apihost"}}/cron/1

</div>
</div>

## Delete a cron job

<div class="api-section">
<div class="api-doc">

This will delete the given cron job. This requires the explicit `cron:delete`
permission.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /cron/:cron

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/cron/1

</div>
</div>
