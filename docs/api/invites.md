[Prev](/api/images) - [Next](/api/keys)

# Invites

* [The invite object](#the-invite-object)
* [List invites for the authenticated user](#list-invites-for-the-authenticated-user)
* [Create an invite](#create-an-invite)
* [Accept an invite](#accept-an-invite)
* [Delete an invite](#delete-an-invite)

## The invite object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the invite.

---

**`namespace_id`** `int` - ID of the namespace the invite was sent for.

---

**`invitee_id`** `int` - ID of the user who received the invite.

---

**`inviter_id`** `int` - ID of the user who sent the invite.

---

**`url`** `string` - The API URL for the invite object itself.

---

**`invitee`** `object` - The [user](/api/user#the-user-object) who received
the invite.

---

**`inviter`** `object` - The [user](/api/user#the-user-object) who sent the
invite.

---

**`namespace`** `object` - The
[namespace](/api/namespaces#the-namespace-object) the invite was for.

</div>
<div class="api-example">

**Object**

    {
        "id": 1,
        "namespace_id": 3,
        "invitee_id": 2,
        "inviter_id": 1,
        "url": "{{index .Vars "apihost"}}/invites/1",
        "invitee": {
            "id": 2,
            "email": "you@example.com",
            "username": "you",
            "created_at": "2006-01-02T15:04:05Z"
        },
        "inviter": {
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
        }
    }

</div>
</div>

## List invites for the authenticated user

<div class="api-section">
<div class="api-doc">

This will list the invites that have been sent to the authenticated user. This
requires the explicit `invite:read` permission.

**Returns**

Returns a list of [invites](/api/invites#the-invite-object).

</div>
<div class="api-example">

**Request**

    GET /invites

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/invites

</div>
</div>

## Create an invite

<div class="api-section">
<div class="api-doc">

This will send an invite to the given user for the given namespace. This
requires the explicit `invite:write` permission. This also requires you to be
the owner of the namespace the invite is being sent for.

**Parameters**

---

**`handle`** `string` - The username or email of the user to invite.

**Returns**

Returns the sent [invite](/api/invites#the-invite-object). It returns an
[error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

**Request**

    POST /n/:user/:path/-/invites

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"handle": "you"}' \
           {{index .Vars "apihost"}}/n/me/djinn/-/invites


    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"handle": "you@example.com"}' \
           {{index .Vars "apihost"}}/n/me/djinn/-/invites

</div>
</div>

## Accept an invite

<div class="api-section">
<div class="api-doc">

This will accept the given invite, and make you a collaborator for the namespace
that invite was for. This requires the explicit `invite:write` permission.

**Returns**

Returns the [invite](/api/invites#the-invite-object).

</div>
<div class="api-example">

**Request**

    PATCH /invites/:invite

**Examples**

    $ curl -X PATCH \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/invites/1   

</div>
</div>

## Delete an invite

<div class="api-section">
<div class="api-doc">

This will delete the given invite. This requires the explicit `invite:delete`
permission. You must also have either sent the given invite, or received it.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /invites/:invite

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/invites/1

</div>
</div>
