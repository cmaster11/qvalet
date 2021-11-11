# HTTP response

You can alter the HTTP response for every listener by using the `httpResponse` plugin.

The customizable elements are:

* HTTP headers
* Status code (defaults to `200`)

You can use templates to customize the fields, and the context of the argument will match the context of the templates
used in the normal listeners, plus an additional `__qvResult` map, which contains the command execution result.

The `__qvResult` map consists of the following fields:

[filename](../../pkg/listener.go ':include :type=code :fragment=exec-command-result')

[filename](../../pkg/routes.go ':include :type=code :fragment=listener-response')

E.g. you can use `.__qvResult.Output`, `__qvResult.Storage`, etc..

NOTE: the plugin will be executed **only** when the command has been executed successfully. If the command returns an
error, there will be a standard response.

## Configuration

[filename](../../pkg/plugin_http_response.go ':include :type=code :fragment=config')

[filename](../../pkg/plugin_http_response.go ':include :type=code :fragment=config-cors')

## Examples

This is an example on how to use the HTTP response plugin:

> Example code at: [`/examples/config.plugin.httpresponse.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.plugin.httpresponse.yaml)

[filename](../../examples/config.plugin.httpresponse.yaml ':include :type=code')

### Redirection

And, another example, which makes use of temporary files to store a redirection target:

> Example code at: [`/examples/config.plugin.httpresponse-file.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.plugin.httpresponse-file.yaml)

[filename](../../examples/config.plugin.httpresponse-file.yaml ':include :type=code')

### CORS

Here you can see how you can use the CORS feature of the plugin, to allow CORS requests:

> Example code at: [`/examples/config.plugin.httpresponse-cors.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.plugin.httpresponse-cors.yaml)

[filename](../../examples/config.plugin.httpresponse-cors.yaml ':include :type=code')

### Custom body

If you want to customize the returned body (instead of the standard wrapped JSON output), you can specify a template to use:

> Example code at: [`/examples/config.plugin.httpresponse-cors.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.plugin.httpresponse-body.yaml)

[filename](../../examples/config.plugin.httpresponse-body.yaml ':include :type=code')