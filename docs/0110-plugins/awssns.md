# AWS SNS

You can create [AWS SNS](https://aws.amazon.com/sns/) subscriptions, and point them to a qValet instance, to automatically confirm the
subscription and process SNS messages. The `awssns` plugin automatically verifies the validity of the messages.

The AWS SNS plugin will add a `/sns` route to your listener, and you can refer to this route when you will add an SNS
subscription.

```
# Before
https://mydomain.com/test

# With SNS plugin
https://mydomain.com/test <- can be used for simulations
https://mydomain.com/test/sns <- automatically decode/support SNS messages
```

Whenever AWS SNS sends a message to the `/sns` endpoint, qValet will inject in your payload the SNS notification
arguments.

E.g. you can access the `Message` field via:

```go-template
My message is: {{ .Message }}
```

All the available SNS notification arguments are defined in the following struct:

[filename](../../pkg/snshttp/notification.go ':include :type=code :fragment=sns-notification')

## Configuration

[filename](../../pkg/plugin_aws_sns.go ':include :type=code :fragment=config')

## Example

This is an example on how to use the AWS SNS plugin:

> Example code at: [`/examples/config.plugin.awssns.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.plugin.awssns.yaml)

[filename](../../examples/config.plugin.awssns.yaml ':include :type=code')
