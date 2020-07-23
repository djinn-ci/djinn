# Variables

* [List variables for the authenticated user](#list-variables-for-the-authenticated-user)
* [Create a variable for the authenticated user](#create-a-variable-for-the-authenticated-user)
* [Get an individual variable](#get-an-individual-variable)
* [Delete a variable](#delete-a-variable)

## List variables for the authenticated user

This will list all of the variables the currently authenticated user has access
to. This requires the explicit `variable:read` permission.

### Request

    GET /variables

**Query Parameters**

| Name     | Type     | Required  | Description                                       |
|----------|----------|-----------|---------------------------------------------------|
| `search` | `string` | N         | Get the variables with keys like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/variables


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/variables?search=ED

### Response

    200 OK
    Content-Length: 2451
    Content-Type: application/json; charset=utf-8
    Link: <https://api.djinn-ci.com/variables?page=1>; rel="prev",
          <https://api.djinn-ci.com/variables?page=3>; rel="next"
    [{
        "id": 1,
        "user_id": 1,
        "namespace_id": null,
        "key": "PGADDR",
        "value": "host=localhost port=5432 dbname=djinn user=djinn password=secret sslmode=disable",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/variables/1",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }, {
        "id": 2,
        "user_id": 1,
        "namespace_id": null,
        "key": "EDITOR",
        "value": "ed",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/variables/2",
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

Nullable fields:
* `namespace_id`

## Create a variable for the authenticated user

This will create a new variable for the authenticated user, this requires the
explicit `variable:write` permission.

### Request

    POST /variables

**Body**

| Name        | Type     | Required  | Description                             |
|-------------|----------|-----------|-----------------------------------------|
| `key`       | `string` | Y         | The key of the variable.                |
| `value`     | `string` | Y         | The value of the variable.              |
| `namespace` | `string` | N         | The namespace to store the variable in. |

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"key": "PGADDR", "value": "host=localhost port=5432"}' \
           https://api.djinn-ci.com/variables


    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"namespace": "djinn", "key": "PGADDR", "value": "host=localhost port=5432"}' \
           https://api.djinn-ci.com/variables

### Response

    201 Created
    Content-Length: 248
    Content-Type: application/json; charset=utf-8
    {
        "id": 1,
        "user_id": 1,
        "namespace_id": null,
        "key": "PGADDR",
        "value": "host=localhost port=5432 dbname=djinn user=djinn password=secret sslmode=disable",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/variables/1",
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
    {"key": ["Key already exists"]}

## Get an individual variable

This will get the given variable, this requires the explicit `variable:write`
permission.

### Request

    GET /variables/:variable

**URI Parameters**

| Name       | Type     | Required  | Description             |
|------------|----------|-----------|-------------------------|
| `variable` | `int`    | Y         | The id of the variable. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/variables/1

### Response

    200 OK
    Content-Length: 248
    Content-Type: application/json; charset=utf-8
    {
        "id": 1,
        "user_id": 1,
        "namespace_id": null,
        "key": "PGADDR",
        "value": "host=localhost port=5432 dbname=djinn user=djinn password=secret sslmode=disable",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/variables/1",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }

Nullable fields:
* `namespace_id`

## Delete a variable

This will delete the given variable, this requires the explicit
`variable:delete` permission.

### Request

    DELETE /variables/:variable

**URI Parameters**

| Name       | Type     | Required  | Description             |
|------------|----------|-----------|-------------------------|
| `variable` | `int`    | Y         | The id of the variable. |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/variables/1

### Response

    204 No Content
