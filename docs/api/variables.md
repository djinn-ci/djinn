[Prev](/api/user) - [Next](/admin)

# Variables

* [The variable object](#the-variable-object)
* [List variables for the authenticated user](#list-variables-for-the-authenticated-user)
* [Create a variable for the authenticated user](#create-a-variable-for-the-authenticated-user)
* [Get an individual variable](#get-an-individual-variable)
* [Delete a variable](#delete-a-variable)

## The variable object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the variable.

---

**`author_id`** `int` - ID of the user who created the variable.

---

**`user_id`** `int` - ID of the user the variable belongs to.

---

**`namespace_id`** `int` `nullable` - ID of the namespace the variable belongs
to if any.

---

**`key`** `string` - The key of the variable.

---

**`value`** `string` - The value of the variable.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the
variable was created.

---

**`url`** `string` - The API URL for the variable object itself.

---

**`author`** `object` `nullable` - The [user](/api/user#the-user-object) who
authored the variable, if any.

---

**`user`** `object` `nullable` - The [user](/api/user#the-user-object) of the
variable, if any.

---

**`namespace`** `object` `nullable` - The
[namespace](/api/namespaces#the-namespace-object) the variable belongs to, if
any.

</div>
<div class="api-example">

**Object**

    {
        "id": 2,
        "user_id": 1,
        "namespace_id": null,
        "key": "PGADDR",
        "value": "host=localhost port=5432 dbname=dev user=root password=secret sslmode=disable",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "{{index .Vars "apihost"}}/variables/2",
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

## List variables for the authenticated user

<div class="api-section">
<div class="api-doc">

This will list all of the variables the currently authenticated user has access
to. This requires the explicit `variable:read` permission.

**Parameters**

---

**`search`** `string` *optional* - Get the variables with keys like the given value.

**Returns**

Returns a list of [variables](/api/variables#the-variable-object). The list
will be paginated to 25 variables per page and will be ordered by the most
recently created object first. If the objects were paginated, then the
pagination information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/variables?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/variables?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /variables

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/variables


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/variables?search=ED

</div>
</div>

## Create a variable for the authenticated user

<div class="api-section">
<div class="api-doc">

This will create a new variable for the authenticated user, this requires the
explicit `variable:write` permission.

**Parameters**

---

**`key`** `string` - The key of the variable.

---

**`value`** `string` - The value of the variable.

---

**`namespace`** `string` *optional* - The namespace to store the variable in.

**Returns**

Returns the [variable](/api/variables#the-variable-object). It returns an
[error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

**Request**

    POST /variables

**Examples**

    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"key": "PGADDR", "value": "host=localhost port=5432"}' \
           {{index .Vars "apihost"}}/variables


    $ curl -X POST \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d '{"namespace": "djinn", "key": "PGADDR", "value": "host=localhost port=5432"}' \
           {{index .Vars "apihost"}}/variables

</div>
</div>

## Get an individual variable

<div class="api-section">
<div class="api-doc">

This will get the given variable, this requires the explicit `variable:write`
permission.

**Returns**

Returns the [variable](/api/variables#the-variable-object).

</div>
<div class="api-example">

**Request**

    GET /variables/:variable

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/variables/1

</div>
</div>

## Delete a variable

<div class="api-section">
<div class="api-doc">

This will delete the given variable, this requires the explicit
`variable:delete` permission.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /variables/:variable

**Examples**

    $ curl -X DELETE \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/variables/1

</div>
</div>>
