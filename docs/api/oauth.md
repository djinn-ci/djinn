[Prev](/api/namespaces) - [Next](/api/objects)

# OAuth

* [Creating an OAuth app](#creating-an-oauth-app)
* [Authorizing an OAuth app](#authorizing-an-oauth-app)
* [Token scopes](#token-scopes)

## Creating an OAuth app

## Authorizing an OAuth app

## Token Scopes

<div class="api-section">
<div class="api-doc">

A scope dictates the sort of access you need to the API. A single scope is made
up of a resource, and the permissions for that resource. There are three
permissions that a resource can have,

**`read`** - Allow a user to get a resource.

---

**`write`** - Allow a user to create or edit a resource.

---

**`delete`** - Allow a user to delete a resource.

each individual scope is represented like so `<resource>:<permission...>`, for
example,

    build:read,write,delete namespace:read,write

The above scope would grant the user the ability to view, create, and kill
builds, and view, create, and edit namespaces.

</div>
</div>
