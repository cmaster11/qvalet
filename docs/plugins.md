# Plugins

This page lists all plugins supported by`go-to-exec`.

## AWS SNS

You can create AWS SNS subscriptions, and point them to a `go-to-exec` instance, to automatically confirm the
subscription and process SNS messages. The AWS SNS plugin automatically verifies the validity of the messages.

The AWS SNS plugin will add a `/sns` route to your listener, and you can refer to this route when you will add an SNS
subscription.

```
# Before
https://mydomain.com/test

# With SNS plugin
https://mydomain.com/test <- can be used for simulations
https://mydomain.com/test/sns <- automatically decode/support SNS messages
```

Whenever AWS SNS sends a message to the `/sns` endpoint, `go-to-exec` will inject in your payload the SNS notification
arguments.

E.g. you can access the `Message` field via:

```go-template
My message is: {{ .Message }}
```

All the available SNS notification arguments are defined in the following struct:

[filename](../pkg/snshttp/notification.go ':include :type=code :fragment=sns-notification')

### Configuration

[filename](../pkg/plugin_aws_sns.go ':include :type=code :fragment=config')

### Example

This is an example on how to use the AWS SNS plugin:

[filename](../examples/config.plugin.awssns.yaml ':include :type=code')

## HTTP response

You can alter the HTTP response for every listener by using the `httpResponse` plugin.

The customizable elements are:

* HTTP headers
* Status code (defaults to `200`)

You can use templates to customize the fields, and the context of the argument will match the context of the templates
used in the normal listeners, plus an additional `__gteResult` map, which contains the command execution result.

The `__gteResult` map consists of the following fields:

[filename](../pkg/listener.go ':include :type=code :fragment=exec-command-result')

[filename](../pkg/routes.go ':include :type=code :fragment=listener-response')

E.g. you can use `.__gteResult.Output`, `__gteResult.Storage`, etc..

NOTE: the plugin will be executed **only** when the command has been executed successfully. If the command returns an
error, there will be a standard response.

### Configuration

[filename](../pkg/plugin_http_response.go ':include :type=code :fragment=config')

### Examples

This is an example on how to use the HTTP response plugin:

[filename](../examples/config.plugin.httpresponse.yaml ':include :type=code')

And, another example, which makes use of temporary files to store a redirection target:

[filename](../examples/config.plugin.httpresponse-file.yaml ':include :type=code')