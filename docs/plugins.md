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

Whenever AWS SNS sends a message to the `/sns` endpoint, `go-to-exec` will inject in your payload the SNS notification arguments.

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

### Configuration

[filename](../pkg/plugin_http_response.go ':include :type=code :fragment=config')

### Example

This is an example on how to use the HTTP response plugin:

[filename](../examples/config.plugin.httpresponse.yaml ':include :type=code')