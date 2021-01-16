[Prev](/user/offline-runner) - [Next](/api/builds)

<div class="api-section">
<div class="api-doc">

# REST API Overview

* [Resources](#resources)
* [Authentication](#authentication)
* [Errors](#errors)

## Resources

Listed below are the resources exposed via the REST API that can be created,
modified, or deleted.

* [Builds](/api/builds)
* [Cron](/api/cron)
* [Images](/api/images)
* [Invites](/api/invites)
* [Keys](/api/keys)
* [Namespaces](/api/namespaces)
* [Objects](/api/objects)
* [Variables](/api/variables)

## Authentication

Authentication to the API is handled via a bearer token that is sent in the
`Authorization` header on each request. This token can either be generated
by the server itself, or generated as part of the OAuth authorization flow
for an application.

For more details on the OAuth authorization flow see
[Authorizing an OAuth app](/api/oauth#authorizing-oauth-apps).

The amount of access a user has to the API is dictate by the scopes of the
bearer token. For more information about token scopes see
[Token scopes](/api/oauth#token-scopes).

## Errors

Errors returned from API endpoints will be JSON encoded payloads. Detailed
below are the different types of errors that can occur,

**Validation Errors**

Validation errors occur when incorrect data is POSTed to an API endpoint. A JSON
object will be returned, where each key in the object will be a field name, and
its value will be an array of strings detailing the errors that occurred. For
example assume we were to submit a build without a manifest then we would get
the following error,

    {
        "manifest": [
            "Build manifest is required",
            "Build manifest, invalid driver specified",
        ]
    }

on validation errors, the HTTP status code will be that of `400 Bad Request`.

**Unprocessable Entities**

These errors occur when trying to submit data to a namespace you do not have
permission to work in. These will look like,

    {
        "message": "..."
    }

and will be sent with the `422 Unprocessable Entity` status code.

**Internal Errors**

Should an internal error occur from the side of the API then the below JSON
object will be sent with an appropriate `4xx` or `5xx` HTTP response code,

    {
        "message": "..."
    }

</div>
</div>
