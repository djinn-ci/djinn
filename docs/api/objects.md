[Prev](/api/oauth) - [Next](/api/user)

# Objects

* [The object object](#the-object-object)
* [List objects for the authenticated user](#list-objects-for-the-authenticated-user)
* [Create an object for the authenticated user](#create-an-object-for-the-authenticated-user)
* [Get an individual object](#get-an-individual-object)
* [Delete an object](#delete-an-object)

## The object object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the object.

---

**`author_id`** `int` - ID of the user who created the object.

---

**`user_id`** `int` - ID of the user the image belongs to.

---

**`namespace_id`** `int` `nullable` - ID of the namespace the image belongs to
if any.

---

**`name`** `string` - The name of the image.

---

**`type`** `string` - The MIME type of the object.

---

**`size`** `int` - The size of the object in bytes.

---

**`md5`** `string` - The MD5 sum hash of the object.

---

**`sha256`** `string` - The SHA256 sum hash of the object.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the object
was created.

---

**`url`** `string` - The API URL for the object itself.

---

**`author`** `object` `nullable` - The [user](/api/user#the-user-object) who
authored the object, if any.

---

**`user`** `object` `nullable` - The [user](/api/user#the-user-object) of the
object, if any.

---

**`namespace`** `object` `nullable` - The
[namespace](/api/namespaces#the-namespace-object) the object belongs to, if any.

</div>
<div class="api-example">

**Object**

    {
        "id": 2,
        "user_id": 1,
        "namespace_id": 3,
        "name": "data"
        "type": "text/plain; charset=utf-8",
        "size": 4097,
        "md5":  "45ff663815a1a57ff3e24f51992238f8",
        "sha256": "2cc0ce967ed630d79f9db9e694e620f19f79afeebd0e1d2928feff773e8a7129",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "{{index .Vars "apihost"}}/objects/1",
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
        }
    }

</div>
</div>

## List objects for the authenticated user

<div class="api-section">
<div class="api-doc">

This will list all of the objects that the currently authenticated user has
access to. This requires the explicit `object:read` permission.

**Parameters**

---

**`search`** `string` *optional* - Get the objects with names like the given value.

**Returns**

Returns a list of [objects](/api/objects#the-object-object). The list will be
paginated to 25 objects per page and will be ordered by the most recently
created object first. If the objects were paginated, then the pagination
information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/objects?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/objects?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /objects

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/objects


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/objects?search=data

</div>
</div>

## Create an object for the authenticated user

<div class="api-section">
<div class="api-doc">

This will create an object for the currentl authenticated user. This requires
the explicit `object:write` permission.

**Parameters**

---

**`name`** `string` - The name of the object to create.

---

**`namespace`** `string` *optional* - The namespace to put the object in.

The contents of the file should be sent in the body of the request. The header
`Content-Type` should be the MIME type of the file being uploaded.

**Returns**

Returns the [object](/api/objects#the-object-object). It returns an
[error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

**Request**

    POST /objects

**Examples**

    $ curl -X POST \
           -H "Content-Type: text/plain; charset=utf-8" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d "@data" \
           {{index .Vars "apihost"}}/objects?name=data

</div>
</div>

## Get an individual object

<div class="api-section">
<div class="api-doc">

This will get the given object. This requires the explicit `object:read`
permission.

**Returns**

Returns the [object](/api/objects#the-object-object). If requested with the
`Accept` header set to `application/octet-stream` then this will download the
object itself.

</div>
<div class="api-example">

**Request**

    GET /objects/:object

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/objects/1

</div>
</div>

## Delete an object

<div class="api-section">
<div class="api-doc">

This will delete the given object. This requires the explicit `object:delete`
permission.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /objects/:object

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/objects/1

</div>
</div>
