# go-to-exec

`gotoexec` listens for HTTP requests and executes commands on demand.

## Configuration example

[filename](../examples/config.simple.yaml ':include :type=code')

Test with:

```bash
# A simple GET request 
curl "http://localhost:7055/hello/id_123?name=Anderson"

# A JSON POST request
curl "http://localhost:7055/hello/id_123" -d '{"name":"Anderson"}' -H 'Content-Type: application/json'
```

## Templates

Commands, args, environment variables and temporary files can be customized using go templates syntax:

```yaml
# Command to run, and list of arguments
command: bash
args:
  - -c
  - |
    echo "Hello {{ .name | upper }}"
env:
  hello: '{{ .name }}'
files:
  dump: |
    {{ dump . }}
```

You can use all functions from the [sprig](https://github.com/Masterminds/sprig) library in the templates, and
some [additional](/template-functions.md) functions.

You can find some advanced use cases in our [examples](/examples) page.

Templates are populated with all parameters from:

* The path: when listening on `/hello/:myParam`, it is possible to use `{{ .myParam }}`.
* The query: `?name=Anderson"` will let you use `{{ .name }}`.
* The request body: all JSON objects are automatically interpreted, given a correct `Content-Type: application/json`
  header.

## Run

Run with:

```bash
# Go version
go run ./cmd --config examples/config.simple.yaml

# Compiled version
gotoexec --config examples/config.simple.yaml
```

Alternatively, the docker image `cmaster11/go-to-exec` is served
on [Docker Hub](https://hub.docker.com/r/cmaster11/go-to-exec).

To run the docker image on e.g. a local Windows machine:

```
docker run -i -t -v "C:/path/to/config.yaml:/mnt/config.yaml" --rm cmaster11/go-to-exec --config /mnt/config.yaml 
```

## Configuration

You can configure:

* A default configuration for all listeners (using the root `defaults` configuration key).
* A configuration for each listener, which will overwrite the relative entries of the default one.

Here are all available configuration entries:

[filename](../pkg/config.go ':include :type=code :fragment=config-docs')

## Temporary files

You can also define temporary files to be written and used at runtime by creating entries in the `files` list.

Example:

```yaml
files:
  tmp1: Hello {{ .name }}
  /opt/tmp2: This is a file in an absolute route!
```

If the key is a relative route, it will be relative to an always-changing temporary location provided by the system (
e.g. `/tmp/gte-1234`).

All temporary files' paths will be accessible also as environment variables (with the `GTE_FILES_` prefix) and template
vars (under the `(gte).files` map).

```
/tmp/key1 -> GTE_FILES_tmp_key1, {{ (gte).files.tmp_key1 }}
key2 -> GTE_FILES_key2, {{ (gte).files.tmp_key2 }}
```

NOTE: in environment variables and in the templates map's keys, all `\W` characters (NOT `a-z`, `A-Z`, `0-9`, `_`) will
be replaced with `_`.

### Example

To see a real-case example, you can look at the following Slack webhook configuration:

[filename](../examples/config.slack.yaml ':include :type=code')

## Authentication

`go-to-exec` provides some basic authentication mechanisms:

* HTTP basic auth
* Api key as query parameter
* Api key as header

Every listener can be configured to accept one or more api keys, so that requests made to that listener will ONLY work
if the api key is in the list.

Here are all available auth configuration entries:

[filename](../pkg/config.go ':include :type=code :fragment=auth-docs')

### Basic auth

The username is configurable via the `httpAuthUsername` config key, and will default to `gte` if none is provided.

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-basic-auth')

### Api key in query string

You can authenticate requests also by passing the api key in the url parameter `__gteApiKey`.

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-query-auth')

### Api key in header

Certain services send webhooks and let you authenticate these webhooks by passing a pre-defined token in a specific HTTP
header (e.g. [GitLab Webhooks](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)).

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-header-auth')

#### HMAC-SHA256

You can use HMAC-SHA256 header validation, which lets you verify the authenticity of the payload (e.g. used by [doorbell.io](https://doorbell.io/)).

[filename](../examples/config.auth.yaml ':include :type=code :fragment=docs-header-auth-hmac-sha256')

## Error handling

In the `defaults` configuration, or for each listener, you can define a `errorHandler` configuration.

The error handler behaves the same way as a listener, but it is triggered only when the relative listener encounters an
execution error (or, when ANY listener encounters an issue, in case the error handler has been defined in the `defaults`
key).

Each error handler will be provided the following arguments on execution:

Argument | Description
---|---
`route` | The failed listener route
`error` | A textual description of the error
`output` | The output of the failed command, if any exists
`args` | The original arguments map passed to the failed listener

You can see how to configure such a handler in the following example, where it will be triggered on every call
to `/hello`:

[filename](../examples/config.onerror.yaml ':include :type=code')

## Trigger condition

You can configure a specific `trigger` condition for every listener. This means that the listener will be invoked only
if the trigger condition is met.

The syntax of the `trigger` field is the same as inside an `if` block of a Go template. All that matters is that the 
`trigger` if-template returns a `true`/`false` result.

```
{{ if eq .name "Wren" }} -> eq .name "Wren"
```

[filename](../examples/config.trigger.yaml ':include :type=code')