# Use cases

Here are some potential use cases for `go-to-exec`!

## Slack message

[filename](../examples/config.slack.yaml ':include :type=code')

## Telegram message (simple)

[filename](../examples/config.telegram.simple.yaml ':include :type=code')

## Telegram message (common vars)

[filename](../examples/config.telegram.common-vars.yaml ':include :type=code')

## Telegram message (action button)

[filename](../examples/config.telegram.action.yaml ':include :type=code')

## Kubernetes restart helper

Blog post: [Restart Kubernetes deployments using HTTP requests](https://cmaster11.medium.com/restart-kubernetes-deployments-using-http-requests-9db21a928c82)

One example use case for `go-to-exec` is to be used to restart deployments on demand from external HTTP calls.

The following sample code can be tested, from inside the Kubernetes cluster, with:

```
curl "http://restart-helper:7055/restart/deployment/a-deployment-name"
```

[filename](../examples/k8s-restart-helper.yaml ':include :type=code')