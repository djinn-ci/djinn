[Prev](/api/namespaces) - [Next](/api/objects)

# OAuth

* [Authorizing an OAuth app](#authorizing-an-oauth-app)
  * [Requesting the identity](#requesting-the-identity)
  * [Redirecting to the app](#redirecting-to-the-app)
* [Token scopes](#token-scopes)

## Authorizing an OAuth app

<div class="api-section">
<div class="api-doc">

Detailed below is the flow users are taken through when authorizing an
application:

### Requesting the identity

    GET {{index .Vars "host"}}/login/oauth/authorize

**Parameters**

**`client_id`** - The client ID you received from Djinn CI when you created a
new app. 

**`redirect_uri`** - The URL in your app where users will be sent once
authenticated.

**`scope`** - A space delimited list of [scopes](/api/oauth#token-scopes).

**`state`** *optional* - A random string used to protect from CSRF attacks.

### Redirecting to the app

Once the user has allowed your app access to their Djinn CI account, they will
be redirected to the `redirect_uri` you set on your app. A temporary `code`
will be passed in `redirect_uri`, this will expire after 10 minutes. If a 
`state` was given during authentication then this will be sent back too, and
should be checked for on your end. If this state does not match then you should
abort immediately.

Extract the `code` from the `redirect_uri` and exchange it,

    POST {{index .Vars "host"}}/login/oauth/token

**Parameters**

**`client_id`** - The client ID you received from Djinn CI when you created a
new app.

**`client_secret`** - The client secret your received from Djinn CI when you
created a new app.

**`code`** - The code you received during the redirect back to your app.

The parameters POSTed to the endpoint should be encoded as a URL string. By
default the response will be URL encoded like so,

    access_token=1a2b3c&token_type=bearer&scope=build:read,write

you can receive a JSON response by setting the `Accept` header to
`application/json`,

    {
        "access_token": "1a2b3c",
        "token_type": "bearer",
        "scope": "build:read,write"
    }

</div>
</div>

## Token Scopes

<div class="api-section">
<div class="api-doc">

A scope dictates the sort of access you need to the API. A single scope is made
up of a resource, and the permissions for that resource. There are three
permissions that a resource can have,

---

**`read`** - Allow a user to get a resource.

---

**`write`** - Allow a user to create or edit a resource.

---

**`delete`** - Allow a user to delete a resource.

---

each individual scope is represented like so `<resource>:<permission...>`, for
example,

    build:read,write,delete namespace:read,write

The above scope would grant the user the ability to view, create, and kill
builds, and view, create, and edit namespaces.

</div>
</div>
