[Prev](/api/objects) - [Next](/api/variables)

# User

* [The user object](#the-user-object)
* [Get the currently authenticated user](#get-the-currently-authenticated-user)

## The user object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the user.

---

**`email`** `string` - The email of the user.

---

**`username`** `string` - The username of the user.

---

**`created_at`** `timestamp` - The RFC3339 formatted string for when the user
created their account.

</div>
<div class="api-example">

**Object**

    {
        "id": 1,
        "email": "me@example.com",
        "username": "me",
        "created_at": "2006-01-02T15:04:05"
    }

</div>
</div>

## Get the currently authenticated user

<div class="api-section">
<div class="api-doc">

This will return the currently authenticated [user](/api/user#the-user-object).

</div>
<div class="api-example">

**Request**

    GET /user

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/user

</div>
