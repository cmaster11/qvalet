# Trigger conditions

You can configure a specific `trigger` condition for every listener. This means that the listener will be invoked only
if the trigger condition is met.

The syntax of the `trigger` field is the same as inside an `if` block of a Go template. All that matters is that the
`trigger` if-template returns a `true`/`false` result.

```
{{ if eq .name "Wren" }} -> eq .name "Wren"
```

[filename](../examples/config.trigger.yaml ':include :type=code')