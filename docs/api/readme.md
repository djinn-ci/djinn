# REST API Overview

* [Resources](#resources)
* [Authentication](#authentication)

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
[Authorizing an OAuth app](/docs/api/oauth.md#authorizing-oauth-apps).

The amount of access a user has to the API is dictate by the scopes of the
bearer token. For more information about token scopes see
[Token scopes](/docs/api/oauth.md#token-scopes).
