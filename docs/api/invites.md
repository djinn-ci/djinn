# Invites

* [List invites for the authenticated user](#list-invites-for-the-authenticated-user)
* [Create an invite](#create-an-invite)
* [Accept an invite](#accept-an-invite)
* [Delete an invite](#delete-an-invite)

## List invites for the authenticated user

This will list the invites that have been sent to the authenticated user. This
requires the explicit `invite:read` permission.

### Request

    GET /invites

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/invites

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

## Create an invite

This will send an invite to the given user for the given namespace. This
requires the explicit `invite:write` permission. This also requires you to be
the owner of the namespace the invite is being sent for.

### Request

    POST /n/:user/:path/-/invites

**URI Parameters**

| Name   | Type     | Required  | Description                       |
|--------|----------|-----------|-----------------------------------|
| `user` | `string` | Y         | The user that owns the namespace. |
| `path` | `string` | Y         | The path of the namespace.        |

**Body**

| Name     | Type     | Required | Description                                  |
|----------|----------|----------|----------------------------------------------|
| `handle` | `string` | Y        | The username or email of the user to invite. |

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"handle": "you"}' \
           https://api.djinn-ci.com/n/me/djinn/-/invites


    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"handle": "you@example.com"}' \
           https://api.djinn-ci.com/n/me/djinn/-/invites

### Response

    201 Created
    Content-Length: 451
    Content-Type: application/json; charset=utf-8
    {

    }

## Accept an invite

This will accept the given invite, and make you a collaborator for the namespace
that invite was for. This requires the explicit `invite:write` permission.

### Request

    PATCH /invites/:invite

**URI Parameters**

| Name     | Type  | Required  | Description           |
|----------|-------|-----------|-----------------------|
| `invite` | `int` | Y         | The id of the invite. |

### Response

    200 OK
    Content-Length: 451
    Content-Type: application/json; charset=utf-8
    {
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
    }

## Delete an invite

This will delete the given invite. This requires the explicit `invite:delete`
permission. You must also have either sent the given invite, or received it.

### Request

    DELETE /invites/:invite

**URI Parameters**

| Name     | Type  | Required  | Description           |
|----------|-------|-----------|-----------------------|
| `invite` | `int` | Y         | The id of the invite. |

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           https://api.djinn-ci.com/invites/1

### Response

    204 No Content
