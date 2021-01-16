[Prev](/api/invites) - [Next](/api/namespaces)

# Keys

* [The key object](#the-key-object)
* [List keys for the authenticated user](#list-keys-for-the-authenticated-user)
* [Create a key for the authenticated user](#create-a-key-for-the-authenticated-user)
* [Get an individual key](#get-an-individual-key)
* [Update a key](#update-a-key)
* [Delete a key](#delete-a-key)

## The key object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the key.

---

**`author_id`** `int` - ID of the user who created the key.

---

**`user_id`** `int` - ID of the user the key belongs to.

---

**`namespace_id`** `int` `nullable` - ID of the namespace the key belongs to,
if any.

---

**`name`** `string` - The name of the key.

---

**`config`** `string` - The key's configuration.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the key
was created.

---

**`updated_at`** `timestamp` - The RFC3339 formatted string at which the key
was updated.

---

**`url`** `string` - The API URL to the cron job object itself.

---

**`author`** `object` `nullable` - The [user](/api/user#the-user-object) who
authored the key, if any.

---

**`user`** `object` `nullable` - The [user](/api/user#the-user-object) of the
key, if any.

---

**`namespace`** `object` `nullable` - The
[namespace](/api/namespaces#the-namespace-object) the key belongs to, if any.

</div>
<div class="api-example">

**Object**

    {
        "id": 1,
        "user_id": 1,
        "namespace_id": 3,
        "name": "id_rsa",
        "config": "UserKnownHostsFile /dev/null",
        "created_at": "2006-01-02T15:04:05Z",
        "updated_at": "2006-01-02T15:04:05Z",
        "url": "{{index .Vars "apihost"}}/api/keys/1",
        "user": {
            "id": 1,
             "email": "me@example.com",
             "username": "me",
             "created_at": "2006-01-02T15:04:05Z"
        }
    }

</div>
</div>

## List keys for the authenticated user

<div class="api-section">
<div class="api-doc">

This will list the keys the currently authenticated user has access to. This
requires the explicit `key:read` permission.

**Parameters**

---

**`search`** `string` - Get the keys with names like the given value.

**Returns**

Returns a list of [keys](/api/keys#the-key-object). The list will be paginated
to 25 keys per page and will be ordered by the most recently created key first.
If the keys were paginated, then the pagination information will be in the
response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/keys?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/keys?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /keys

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/keys


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/keys?search=id_rsa

</div>
</div>

## Create a key for the authenticated user

<div class="api-section">
<div class="api-doc">

This will create a ket for the authenticated user. This requires the explicit
`key:write` permission.

**Parameters**

---

**`name`** `string` - The name of the key.

---

**`key`** `string` - The private key.

---

**`config`** `string` - The config for the key. 

---

**`namespace`** `string` - The namespace to store the key in.

**Returns**

Returns the [key](/api/keys#the-key-object). It returns an [error](/api#errors)
if any of the parameters are invalid, or if an internal error occurs.

</div>
<div class="api-example">

**Request**

    POST /keys

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"name": "id_rsa", "key": "-----BEGIN..."}' \
           {{index .Vars "apihost"}}/keys/1

</div>
</div>

## Get an individual key

<div class="api-section">
<div class="api-doc">

This will get the key with the given id. This requires the explicit `key:read`
permission.

**Returns**

Returns the [key](/api/keys#the-key-object).

</div>
<div class="api-example">

**Request**

    GET /keys/:key

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/keys/1

</div>
</div>

## Update a key

<div class="api-section">
<div class="api-doc">

This will update the given key, this requires the explicit `key:write`
permission.

**Parameters**

---

**`name`** `string` *optional* - The name of the key.

---

**`config`** `string` *optional* - The config for the key.

---

**`namespace`** `string` *optional* - The namespace to store the key in.

</div>
<div class="api-example">

**Request**

    PATCH /keys/:key

**Examples**

    $ curl -X PATCH \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"config": "UserKnownHostsFile /dev/null"}' \
           {{index .Vars "apihost"}}/keys/1

</div>
</div>

## Delete a key

<div class="api-section">
<div class="api-doc">

This will delete the given key, this requires the explicit `key:delete`
permission.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /keys/:key

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/keys/1

</div>
</div>
