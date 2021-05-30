# go-to-exec

`gotoexec` listens for HTTP requests and executes commands on demand.

Commands can be customized using go templates syntax:

```yaml
# Command to run and list of arguments
command: bash
args:
  - -c
  - |
    echo "Hello {{ .name | upper }}"
```

You can use all functions from the [sprig](https://github.com/Masterminds/sprig) library in the templates.

Templates are populated with all parameters from:

* The path: when listening on `/hello/:myParam`, it is possible to use `{{ .myParam }}`.
* The query: `?name=Anderson"` will let you use `{{ .name }}`.
* The request body: all JSON objects are automatically interpreted, given a correct `Content-Type: application/json` header.

## Run and test

Run with:

```bash
# Go version
go run . --config config.yaml

# Compiled version
gotoexec --config config.yaml
```

Alternatively, the docker image `cmaster11/go-to-exec` is served on [Docker Hub](https://hub.docker.com/r/cmaster11/go-to-exec).

Test with:

```bash
# A simple GET request 
curl "http://localhost:7055/hello/id_123?name=Anderson"

# A JSON POST request
curl "http://localhost:7055/hello/id_123" -d '{"name":"Anderson"}' -H 'Content-Type: application/json'
```

## Configuration struct

[filename](config.go ':include :type=code :fragment=config-docs')

## Configuration example

[filename](config.yaml ':include :type=code')