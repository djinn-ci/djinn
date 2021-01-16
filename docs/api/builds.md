[Prev](/api) - [Next](/api/cron)

# Builds

* [The build object](#the-build-object)
* [The trigger object](#the-trigger-object)
* [The job object](#the-job-object)
* [The artifact object](#the-artifact-object)
* [The tag object](#the-tag-object)
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

## The build object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the build.

---

**`user_id`** `int` - ID of the user the build belongs to.

---

**`namespace_id`** `int` `nullable` - ID of the namespace the build belongs
to if any.

---

**`number`** `int` - Number of the build for the user who submitted it.

---

**`manifest`** `string` - The manifest of the build.

---

**`status`** `enum` - The status of the build, will be one of, `queued`,
`running`, `passed`, `passed_with_failures`, `failed`, `killed`, or `timed_out`.

---

**`output`** `string` `nullable` - The output of the build if any.

---

**`tags`** `string[]` - The list of tags on the build.

---

**`created_at`** `timestamp` -  The RFC3339 formatted string at which the build
was created.

---

**`started_at`** `timestamp` `nullable` - The RFC3339 formatted string at
which the build was started.

---

**`finished_at`** `timestamp` `nullable` - The RFC3339 formatted string at
which the build finished.

---

**`url`** `string` - The API URL to the build object itself.

---

**`objects_url`** `string` - The API URL to the build's objects.

---

**`variables_url`** `string` - The API URL to the build's variables.

---

**`jobs_url`** `string` - The API URL to the build's jobs.

---

**`artifacts_url`** `string` - The API URL to the build's artifacts.

---

**`tags_url`** `string` - The API URL to the build's tags.

---

**`user`** `object` `nullable` - The [user](/api/user#the-user-object) of the
build, if any.

---

**`namespace`** `object` `nullable` - The
[namespace](/api/namespaces#the-namespace-object) namespace of the build, if
any.

---

**`trigger`** `object` `nullable` - The
[trigger](/api/builds#the-trigger-object) of the build, if any.

</div>

<div class="api-example">

**Object**

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

</div>
</div>

## The trigger object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`type`** `enum` - The type of trigger for the build, one of `manual`, `push`,
`pull`, or `schedule`.

---

**`comment`** `string` - The comment associated with the build.

---

**`data`** `object` - A `string:string` object of the data about the trigger,
such as who authored it, and any commit information associated with it, if this
was a `push`, or `pull` trigger.

</div>

<div class="api-example">

**Object**

    {
        "type": "manual",
        "comment": "",
        "data": {
            "email": "me@example.com",
            "username": "me"
        }
    }

</div>
</div>

## The job object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the job.

---

**`build_id`** `int` - ID of the build the job belongs to.

---

**`name`** `string` - The name of the job.

---

**`commands`** `string` - The commands for the job.

---

**`status`** `enum` - The status of the job, will be one of, `queued`,
`running`, `passed`, `passed_with_failures`, `failed`, `killed`, or `timed_out`.

---

**`output`** `string` `nullable` - The output of the job, if any.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the job
was created.

---

**`started_at`** `timestamp` `nullable` - The RFC3339 formatted string at which
the job was started.

---

**`finished_at`** `timestamp` `nullable` - The RFC3339 formatted string at
which the job finished.

---

**`url`** `string` - The API URL to the job object itself.

---

**`build`** `object` - The [build](/api/builds#the-build-object) of the job, if
any.

</div>

<div class="api-example">

**Object**

    {
        "id": 4,
        "build_id": 3,
        "name": "create driver",
        "commands": "",
        "status": "queued",
        "output": null,
        "created_at": "2006-01-02T15:04:05Z",
        "started_at": null,
        "finished_at": null,
        "url": "{{index .Vars "apihost"}}/b/me/3/jobs/4"
    }

</div>
</div>

## The artifact object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the job.

---

**`build_id`** `int` - ID of the build the artifact belongs to.

---

**`job_id`** `int` - ID of the job the artifact belongs to.

---

**`source`** `string` - The original name of the artifact in the build
environment.

---

**`name`** `string` - The name of the artifact it was collected as.

---

**`size`** `int` `nullable` - The size of the artifact, if any.

---

**`md5`** `string` `nullable` - The MD5 hash of the artifact, if any.

---

**`sha256`** `string` `nullable` - The SHA256 hash of the artifact, if any.

---

**`created_at`** `timestamp` - The RFC3339 formatted time at which the artifact
was created.

---

**`url`** `string` - The API URL to the artifact object iself.

---

**`build`** `object` - The [build](/api/builds) of the artifact, if any.

</div>
<div class="api-example">

**Object**

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
        "url": "{{index .Vars "apihost"}}/b/me/3/artifacts/1"
    }

</div>
</div>

## The tag object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the tag.

---

**`user_id`** `int` - ID of the user who created the tag.

---

**`build_id`** `int` - ID of the build the tag belongs to.

---

**`name`** `string` - The name of the tag.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the tag
was created.

---

**`url`** `string` - The API URL to the tag object itself.

---

**`user`** `object` - The [user](/api/user#the-user-object) of the tag, if any.

---

**`build`** `object` - The [build](/api/builds#the-build-object) of the build
if any.

</div>
<div class="api-example">

**Object**

    {
        "id": 3,
        "user_id": 1,
        "build_id": 3,
        "name": "centos/7",
        "created_at": "2006-01-02T15:04:05Z"
        "url": "{{index .Vars "apihost"}}/b/me/3/tags/3",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }

</div>
</div>

## List builds for the authenticated user

<div class="api-section">
<div class="api-doc">

This will list all of the builds that the currently authenticated user has
access to. This requires the explicit `build:read` permission for the user.

**Parameters**

---

**`tag`** `string` *optional* - Get the builds with the given tag name.

---

**`search`** `string` *optional* - Get the builds with tags like the given value.

---

**`status`** `string` *optional* - Get the builds with the given status.

**Returns**

Returns a list of [builds](/api/builds#the-build-object). The list will be
paginated to 25 builds per page and will be ordered by the most recently
submitted builds first. If the builds were paginated, then the pagination
information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/builds?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/builds?page=3>; rel="next"

</div>

<div class="api-example">

**Request**

    GET /builds

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/builds


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/builds?search=go&status=finished

</div>
</div>

## Submit a build for the authenticated user

<div class="api-section">
<div class="api-doc">

This will submit a new build to the server for the currently authenticated user.
This requires the explicit `build:write` permission.

**Parameters**

---

**`manifest`** `string` - The YAML formatted build manifest.

---

**`comment`** `string` *optional* - The build's comment.

---

**`tags`** `string[]` *optional* - A list of tags to attach to the build.

**Returns**

Returns the [build](/api/builds#the-build-object) will be returned. It returns
an [error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

**Request**

    POST /builds

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"manifest":"namespace: djinn\ndriver:\n  image: centos/7\n  type: qemu\nenv:\n- LOCALE=en_GB.UTF-8\nobjects:\n- data => data\nstages:\n- clean\njobs:\n- stage: clean\n  commands:\n  - tr -d '0-9' data > data.cleaned\n  artifacts:\n  - data.cleaned => data.cleaned"}' \
           {{index .Vars "apihost"}}/builds


    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"manifest":"namespace: djinn\ndriver:\n  image: centos/7\n  type: qemu\nenv:\n- LOCALE=en_GB.UTF-8\nobjects:\n- data => data\nstages:\n- clean\njobs:\n- stage: clean\n  commands:\n  - tr -d '0-9' data > data.cleaned\n  artifacts:\n  - data.cleaned => data.cleaned", "tags":["tag1"]}' \
           {{index .Vars "apihost"}}/builds

</div>
</div>

## Get a build

<div class="api-section">
<div class="api-doc">

This will get the build by the given `:user`, with the given `:id`. This
requires the explicit `build:read` permission.

**Returns**

Returns the [build](/api/builds#the-build-object).

</div>

<div class="api-example">

**Request**

    GET /b/:user/:id

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3

</div>
</div>

## Get a build's objects

<div class="api-section">
<div class="api-doc">

This will return a list of all the objects that have been placed, or will be
placed on the build. This requires the explicit `build:read` permission.

**Returns**

Returns a list of [objects](/api/objects#the-object-object) for the given build.

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/objects

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/objects

</div>
</div>

## Get a build's variables

<div class="api-section">
<div class="api-doc">

This will return a list of all the variables that have been set for the build,
either set from the build manifest itself, or set via the variables resource.
This requires the explicit `build:read` permission.

**Returns**

Returns a list of [variables](/api/variables#the-variable-object) for the given
build.

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/variables

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/variables

</div>
</div>

## Get a build's jobs

<div class="api-section">
<div class="api-doc">

This will return a list of all the jobs that are part of the build. This
requires the explicit `build:read` permission.

**Returns**

Returns a list of [jobs](/api/builds#the-job-object) for the given build.

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/jobs

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/jobs

</div>
</div>

## Get an individual build job

<div class="api-section">
<div class="api-doc">

This will return an individual job for the given build. Ths requires the
explicit `build:read` permission.

**Returns**

Returns the given [job](/api/builds#the-job-object).

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/jobs/:job_id

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/jobs/5

</div>
</div>

## Get a build's artifacts

<div class="api-section">
<div class="api-doc">

This will list the artifacts that have been collected from the given build. This
requires the explicit `build:read` permission.

**Returns**

Returns a list of [artifacts](/api/builds#the-artifact-object) for the given
build.

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/artifacts

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/artifacts

</div>
</div>

## Get an individual build artifact

<div class="api-section">
<div class="api-doc">

This will get an individual artifact for the given build. This requires the
explicit `build:read` permission.

**Returns**

Returns the given [artifact](/api/builds#the-artifact-object).

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/artifacts/:artifact_id

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/artifacts/1

</div>
</div>

## Get a build's tags

<div class="api-section">
<div class="api-doc">

This will list the tags set on the given build. This requires the explicit
`build:read` permission.

**Returns**

Returns a list of [tags](/api/builds#the-tag-object) for the given build.

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/tags

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/tags

</div>
</div>

## Get an individual tag from a build

<div class="api-section">
<div class="api-doc">

This will get the given tag from the given build. This requires the explicit
`build:read` permission.

**Returns**

Returns the given [tag](/api/builds#the-tag-objects).

</div>
<div class="api-example">

**Request**

    GET /b/:user/:id/tags/:tag_id

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/tags/4

</div>
</div>

## Add tags to a build

<div class="api-section">
<div class="api-doc">

This will add the given tags to the given build. This requires the explicit
`build:write` permission.

**Parameters**

This expects a JSON array of strings in the request body.

**Returns**

Returns a list of [tags](/api/builds#the-tag-object). It returns an
[error](/api#errors) if an internal error occurs.

</div>
<div class="api-example">

**Request**

    POST /b/:user/:id/tags

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '["tag1", "tag2", "tag3"]' \
           {{index .Vars "apihost"}}/b/me/3/tags

</div>
</div>

## Remove tags from a build

<div class="api-section">
<div class="api-doc">

This will remove the given tag from a given build. This requires the explicit
`build:delete` permission.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /b/:user/:id/tags/:tag_id

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3/tags/5

</div>
</div>

## Kill a build

<div class="api-section">
<div class="api-doc">

This will kill a build that is running. This requires the explicit
`build:delete` permission.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /b/:user/:id

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/b/me/3

</div>
</div>
