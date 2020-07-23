# Keys

* [List keys for the authenticated user](#list-keys-for-the-authenticated-user)
* [Create a key for the authenticated user](#create-a-key-for-the-authenticated-user)
* [Get an individual key](#get-an-individual-key)
* [Update a key](#update-a-key)
* [Delete a key](#delete-a-key)

## List keys for the authenticated user

This will list the keys the currently authenticated user has access to. This
requires the explicit `key:read` permission.

### Request

    GET /keys

**Query Parameters**

| Name     | Type     | Required  | Description                                   |
|----------|----------|-----------|-----------------------------------------------|
| `search` | `string` | N         | Get the keys with names like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/keys


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/keys?search=id_rsa

### Response

    200 OK
    Content-Length: 306
    Content-Length: application/json; charset=utf-8
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
        }
    }]

## Create a key for the authenticated user

This will create a ket for the authenticated user. This requires the explicit
`key:write` permission.

### Request

    POST /keys

**Body**

| Name        | Type     | Required  | Description                        |
|-------------|----------|-----------|------------------------------------|
| `name`      | `string` | Y         | The name of the key.               |
| `key`       | `string` | Y         | The private key.                   |
| `config`    | `string` | N         | The config for the key.            |
| `namespace` | `string` | N         | The namespace to store the key in. |

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "id_rsa", "key": "-----BEGIN..."}' \
           https://api.djinn-ci.com/keys/1

### Response

    201 Created
    Content-Length: 273
    Content-Type: application/json; charset=utf-8
    {
        "id": 1,
        "user_id": 1,
        "namespace_id": 3,
        "name": "id_rsa",
        "config": "",
        "created_at": "2006-01-02T15:04:05Z",
        "updated_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/api/keys/1",
        "user": {
            "id": 1,
             "email": "me@example.com",
             "username": "me",
             "created_at": "2006-01-02T15:04:05Z"
        }
    }

Nullable fields:
* `namespace_id`

If any of the required request paramrters are missing, or of an invalid value,
then a `400 Bad Request` response is sent back, detailing the errors for each
parameter.

    400 Bad Request
    Content-Length: 48
    Content-Type: application/json; charset=utf-8
    {"key": ["Key is not valid"]}

## Get an individual key

This will get the key with the given id. This requires the explicit `key:read`
permission.

### Request

    GET /keys/:key

**URI Parameters**

| Name  | Type  | Required  | Description        |
|-------|-------|-----------|--------------------|
| `key` | `int` | Y         | The id of the key. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/keys/1

### Response

    200 OK
    Content-Length: 273
    Content-Type: application/json; charset=utf-8
    {
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
        }
    }

Nullable fields:
* `namespace_id`

## Update a key

This will update the given key, this requires the explicit `key:write`
permission.

### Request

    PATCH /keys/:key

**URI Parameters**

| Name  | Type  | Required  | Description        |
|-------|-------|-----------|--------------------|
| `key` | `int` | Y         | The id of the key. |

**Body**

| Name        | Type     | Required  | Description                        |
|-------------|----------|-----------|------------------------------------|
| `name`      | `string` | Y         | The name of the key.               |
| `config`    | `string` | N         | The config for the key.            |
| `namespace` | `string` | N         | The namespace to store the key in. |

**Examples**

    $ curl -X PATCH \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"config": "UserKnownHostsFile /dev/null"}' \
           https://api.djinn-ci.com/keys/1

### Response

    200 OK
    Content-Length: 273
    Content-Type: application/json; charset=utf-8
    {
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
        }
    }

## Delete a key

This will delete the given key, this requires the explicit `key:delete`
permission.

### Request

    DELETE /keys/:key

**URI Parameters**

| Name  | Type  | Required  | Description        |
|-------|-------|-----------|--------------------|
| `key` | `int` | Y         | The id of the key. |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/keys/1

### Response

    204 No Content
