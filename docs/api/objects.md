# Objects

* [List objects for the authenticated user](#list-objects-for-the-authenticated-user)
* [Create an object for the authenticated user](#create-an-object-for-the-authenticated-user)
* [Get an individual object](#get-an-individual-object)
* [Delete an object](#delete-an-object)

## List objects for the authenticated user

This will list all of the objects that the currently authenticated user has
access to. This requires the explicit `object:read` permission.

### Request

    GET /objects

**Query Parameters**

| Name     | Type     | Required  | Description                                      |
|----------|----------|-----------|--------------------------------------------------|
| `search` | `string` | N         | Get the objects with names like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/objects


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/objects?search=data

### Response

    200 OK
    Content-Type: application/json; charset=utf-8
    [{
        "id": 1,
        "user_id": 1,
        "namespace_id": null,
        "name": "file",
        "type": "text/plain; charset=utf-8",
        "size": 4097,
        "md5":  "45ff663815a1a57ff3e24f51992238f8",
        "sha256": "2cc0ce967ed630d79f9db9e694e620f19f79afeebd0e1d2928feff773e8a7129",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/objects/1",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }, {
        "id": 2,
        "user_id": 1,
        "namespace_id": 3,
        "name": "data"
        "type": "text/plain; charset=utf-8",
        "size": 4097,
        "md5":  "45ff663815a1a57ff3e24f51992238f8",
        "sha256": "2cc0ce967ed630d79f9db9e694e620f19f79afeebd0e1d2928feff773e8a7129",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/objects/1",
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

## Create an object for the authenticated user

This will create an object for the currentl authenticated user. This requires
the explicit `object:write` permission.

### Request

    POST /objects

**Query Parameters**

| Name        | Type     | Required  | Description                         |
|-------------|----------|-----------|-------------------------------------|
| `name`      | `string` | Y         | The name of the object to create.   |
| `namespace` | `string` | N         | The namespace to put the object in. |

**Body**

The body of this request should be the contents of the object file being
created. The header `Content-Type` should be the MIME type of the file being
uploaded.

**Examples**

    $ curl -X POST \
           -H "Content-Type: text/plain; charset=utf-8" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d "@data" \
           https://api.djinn-ci.com/objects?name=data

### Response

    201 Created
    Content-Type: application/json; charset=utf-8
    {

    }

## Get an individual object

This will get the given object. This requires the explicit `object:read`
permission.

### Request

    GET /objects/:object

**URI Parameters**

| Name     | Type  | Required  | Description           |
|----------|-------|-----------|-----------------------|
| `object` | `int` | Y         | The id of the object. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/objects/1

### Response

    200 OK
    Content-Type: application/json; charset=utf-8
    {
        "id": 1,
        "user_id": 1,
        "namespace_id": null,
        "name": "file",
        "type": "text/plain; charset=utf-8",
        "size": 4097,
        "md5":  "45ff663815a1a57ff3e24f51992238f8",
        "sha256": "2cc0ce967ed630d79f9db9e694e620f19f79afeebd0e1d2928feff773e8a7129",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/objects/1",
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
    {"name": ["Name already exists"]}

## Delete an object

This will delete the given object. This requires the explicit `object:delete`
permission.

### Request

    DELETE /objects/:object

**URI Parameters**

| Name     | Type  | Required  | Description           |
|----------|-------|-----------|-----------------------|
| `object` | `int` | Y         | The id of the object. |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/objects/1

### Response

    204 No Content
