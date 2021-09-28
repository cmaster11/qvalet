# Templates

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

You can find some advanced use cases in our [use cases](/use-cases) page and in
our [examples](https://github.com/cmaster11/go-to-exec/tree/main/examples) folder.

Templates are populated with all parameters from:

* The path: when listening on `/hello/:myParam`, it is possible to use `{{ .myParam }}`.
* The query: `?name=Anderson"` will let you use `{{ .name }}`.
* The body; `go-to-exec` accepts and will parse the following content types:

| Format | Content types |
| --- | --- |
| JSON | `application/json`, `text/plain`, no content type defined |
| Form | `application/x-www-form-urlencoded`, `multipart/form-data` |
| YAML | `application/x-yaml`, `application/yaml`, `text/yaml`, `text/x-yaml` |

You can then use any fields of these objects in your templates.

* The headers: all request headers will be copied into the `__gteHeaders` map, with their keys lower-cased:

```
{{ .__gteHeaders.x-my-token }}
```

## Array payload

A special case applies when a listener receives an array payload (JSON/YAML content type). In this case, the processed
payload will be an object, where there is one key for each array index (`0, 1, 2 -> "0", "1", "2"`), and the
key `__gtePayloadArrayLength` will contain the length of the original array. This is an example:

[filename](../examples/config.simple.array.yaml ':include :type=code')