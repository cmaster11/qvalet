# Authentication

`go-to-exec` provides some basic authentication mechanisms:

* HTTP basic auth
* Api key as query parameter
* Api key as header

Every listener can be configured to accept one or more api keys, so that requests made to that listener will ONLY work
if the api key is in the list.

Here are all available auth configuration entries:

[filename](../pkg/auth.go ':include :type=code :fragment=auth-docs')

## Basic auth

The username is configurable via the `httpAuthUsername` config key, and will default to `gte` if none is provided.

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-basic-auth')

And, basic authentication using environment variables:

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-basic-auth-env')

## Api key in query string

You can authenticate requests also by passing the api key in the url parameter `__gteApiKey`.

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-query-auth')

## Api key in header

Certain services send webhooks and let you authenticate these webhooks by passing a pre-defined token in a specific HTTP
header (e.g. [GitLab Webhooks](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)).

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-header-auth')

### HMAC-SHA256

You can use HMAC-SHA256 header validation, which lets you verify the authenticity of the payload (e.g. used
by [doorbell.io](https://doorbell.io/)).

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-header-auth-hmac-sha256')

### `transform`

Certain services (
e.g. [GitHub](https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks)) will send
authentication headers with additional content, like:

```
X-Hub-Signature-256: sha256=53dac1b832da1a9c46285c9ddb7af65d139199690e62abd628063a6fbd697394
```

`go-to-exec` generates a plain hash, without prefixes. To be able to match the two, you can use the `transform` field:

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-header-auth-hmac-sha256-transform')
