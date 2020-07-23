# REST API Overview

* [Resources](#resources)
* [Authentication](#authentication)
  * [Scopes](#scopes)

## Resources

Listed below are the resources exposed via the REST API that can be created,
modified, or deleted.

* [Builds](/docs/api/builds.md)
* [Namespaces](/docs/api/namespaces.md)
* [Objects](/docs/api/objects.md)
* [Keys](/docs/api/keys.md)
* [Variables](/docs/api/variables.md)
* [Images](/docs/api/images.md)
* [Invites](/docs/api/invites.md)

## Authentication

Authentication to the API is handled via a bearer token that is sent in the
`Authorization` header on each request. This token can either be generated
by the server itself, or generated as part of the OAuth authorization flow
for an application.

For more details on the OAuth authorization flow see
[Authorizing OAuth Apps](/docs/api/oauth.md#authorizing-oauth-apps).

### Scopes

The bearer token that is used for authenticating incoming requests to the API
will have a set of scopes against them. These scopes will dictate what the user
can do with a resource. Each resource has three permissions,

* `read` - Get a resource
* `write` - Create or modify a resource
* `delete` - Delete a resource

when a resource and permission are put together you get a single scope, for
example `build:read,write`. This scope would grant the user the ability to
create a build, and get a build.

For more information about token scopes see
[Token Scopes](/docs/api/oauth.md#token-scopes).
