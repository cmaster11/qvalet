# Error handling

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

> Example code at: [`/examples/config.onerror.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.onerror.yaml)

[filename](../examples/config.onerror.yaml ':include :type=code')
