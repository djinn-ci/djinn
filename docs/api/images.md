# Images

* [List images for the authenticated user](#list-images-for-the-authenticated-user)
* [Create an image for the authenticated user](#create-an-image-for-the-authenticated-user)
* [Get an individual image](#get-an-individual-image)
* [Delete an image](#delete-an-image)

## List images for the authenticated user

This will list all of the images that the currently authenticated user has
access to. This requires the explicit `image:read` permission.

### Request

    GET /images

**Parameters**

| Name     | Type     | Required  | Description                                     |
|----------|----------|-----------|-------------------------------------------------|
| `search` | `string` | N         | Get the images with names like the given value. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/images


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/images?search=go-dev

### Response

    200 OK
    Content-Length: 1226
    Content-Type: application/json; charset=utf-8
    [{
        "id": 1,
        "user_id": 1,
        "namespace_id": null,
        "name": "go-dev",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/images/1",
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
        "name": "go-dev",
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

Nullable fields:

* `namespace_id`

## Create an image for the authenticated user

This will create an image for the currently authenticated user. This requires
the explicit `image:write` permission.

### Request

    POST /images

**Parameters**

| Name        | Type     | Required  | Description                        |
|-------------|----------|-----------|------------------------------------|
| `name`      | `string` | Y         | The name of the image to create.   |
| `namespace` | `string` | N         | The namespace to put the image in. |

The body of this request should be the contents of the image file being created.

**Examples**

    $ curl -X POST \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d "@alpine.qcow2" \
           https://api.djinn-ci.com/images?name=alpine


    $ curl -X POST \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d "@alpine.qcow2" \
           https://api.djinn-ci.com/images?name=alpine&namespace=djinn

### Response

    201 Created
    Content-Length: 236
    Content-Type: application/json; charset=utf-8
    {
        "id": 3,
        "user_id": 1,
        "namespace_id": null,
        "name": "alpine",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/images/3",
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

## Get an individual image

This will get the given image, this requires the explicit `image:read`
permission.

### Request

    GET /images/:image

**Parameters**

| Name    | Type  | Required | Description          |
|---------|-------|----------|----------------------|
| `image` | `int` | Y        | The id of the image. |

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/images/3

    $ curl -X GET \
           -H "Accept: application/octet-stream" \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/images/3

### Response

    200 OK
    Content-Length: 230
    Content-Type: application/json; charset=utf-8
    {
        "id": 3,
        "user_id": 1,
        "namespace_id": null,
        "name": "alpine",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "https://api.djinn-ci.com/images/3",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }

If the `Accept` header is set to `application/octet-stream` then the response
body will be the contents of the image file.

## Delete an image

This will delete the given image, this requires the explicit `image:delete`
permission.

### Request

    DELETE /images/:image

**Parameters**

| Name    | Type  | Required | Description          |
|---------|-------|----------|----------------------|
| `image` | `int` | Y        | The id of the image. |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/images/3

### Response

    204 No Content
