# REST API Overview

* [Resources](#resources)
* [Authentication](#authentication)

## Resources

Listed below are the resources exposed via the REST API that can be created,
modified, or deleted.

* [Builds](/api/builds)
* [Namespaces](/api/namespaces)
* [Objects](/api/objects)
* [Keys](/api/keys)
* [Variables](/api/variables)
* [Images](/api/images)
* [Invites](/api/invites)

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