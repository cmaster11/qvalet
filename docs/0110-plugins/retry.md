# Retry

There are cases where you may want to retry a command execution (let's imagine when you have temporary errors, like
rate-limiting ones). For this purpose you can use the `retry` plugin!

After every command execution, you can have one or more checks to verify if you need to retry the whole flow.

Also, when evaluating the `condition` and `delay` fields, you will be able to access an additional `__qvRetry` map, which can
help you figure out if and when to retry:

[filename](../../pkg/plugin_retry.go ':include :type=code :fragment=retry-payload')

Here is the structure of `ExecCommandResult`:

[filename](../../pkg/listener.go ':include :type=code :fragment=exec-command-result')

NOTE: the `condition` field uses the syntax of an [if-template](/0900-appendix/if-templates.md).

## Configuration

[filename](../../pkg/plugin_retry.go ':include :type=code :fragment=config')

## Examples

> Example code at: [`/examples/config.plugin.retry.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.plugin.retry.yaml)

This is a simple example on how to use the retry plugin:

[filename](../../examples/config.plugin.retry.yaml ':include :type=code :fragment=docs-retry-simple')

This is an example on how to react to a 429 status code after performing a curl request:

[filename](../../examples/config.plugin.retry.yaml ':include :type=code :fragment=docs-retry-429')
