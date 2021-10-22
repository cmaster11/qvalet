# If-templates

The syntax of an "if-template" (which you can see e.g in the [trigger condition](/0080-trigger-conditions.md), or in
the [retry plugin condition](/0110-plugins/retry.md)) is the same as inside an `if` block of a Go template.

All that matters is that the if-template returns a `true`/`false` result.

```
{{ if eq .name "Wren" }} -> eq .name "Wren"
```