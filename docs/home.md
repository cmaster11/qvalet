# go-to-exec

`gotoexec` listens for HTTP requests and executes commands on demand.

## Run

Run with:

```bash
# Go version
go run ./cmd --config examples/config.simple.yaml

# Compiled version
gotoexec --config examples/config.simple.yaml
```

Alternatively, the docker image `cmaster11/go-to-exec` is served on [Docker Hub](https://hub.docker.com/r/cmaster11/go-to-exec).

To run the docker image on e.g. a local Windows machine:

```
docker run -i -t -v "C:/path/to/config.yaml:/mnt/config.yaml" --rm cmaster11/go-to-exec --config /mnt/config.yaml 
```

## Templates

Commands can be customized using go templates syntax:

```yaml
# Command to run, and list of arguments
command: bash
args:
  - -c
  - |
    echo "Hello {{ .name | upper }}"
```

You can use all functions from the [sprig](https://github.com/Masterminds/sprig) library in the templates, and some [additional](/template-functions.md) functions.

You can find some advanced use cases in our [examples](/examples) page.

Templates are populated with all parameters from:

* The path: when listening on `/hello/:myParam`, it is possible to use `{{ .myParam }}`.
* The query: `?name=Anderson"` will let you use `{{ .name }}`.
* The request body: all JSON objects are automatically interpreted, given a correct `Content-Type: application/json` header.

## Configuration example

[filename](../examples/config.simple.yaml ':include :type=code')

Test with:

```bash
# A simple GET request 
curl "http://localhost:7055/hello/id_123?name=Anderson"

# A JSON POST request
curl "http://localhost:7055/hello/id_123" -d '{"name":"Anderson"}' -H 'Content-Type: application/json'
```

## Configuration struct

[filename](../gotoexec/config.go ':include :type=code :fragment=config-docs')

## Temporary files

You can also define temporary files to be written and used at runtime by creating entries in the `Files` list.

Example:

```yaml
files:
  
    tmp1:
        Hello {{ .name }}
    
    /opt/tmp2:
        This is a file in an absolute route!
```

If the key is a relative route, it will be relative to an always-changing temporary location provided by the system (e.g. `/tmp/gte-1234`).

All temporary files' paths will be accessible also as environment variables (with the `GTE_FILES_` prefix) and template vars (under the `(gte).files` map).

```
/tmp/key1 -> GTE_FILES_tmp_key1, {{ (gte).files.tmp_key1 }}
key2 -> GTE_FILES_key2, {{ (gte).files.tmp_key2 }}
```

NOTE: in environment variables and in the templates map's keys, all `\W` characters (NOT `a-z`, `A-Z`, `0-9`, `_`) will be replaced with `_`.

### Example

To see a real-case example, you can look at the following Slack webhook configuration:

[filename](../examples/config.slack.yaml ':include :type=code')