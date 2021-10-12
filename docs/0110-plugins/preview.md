# Preview

There are cases where you may want to preview the command you want to execute, think for example about any critical
commands you may want to run once or check in advance.

For this purpose you can use the `preview` plugin, which will add a `/preview` route to your listener(s). Using this
route, you will be able to see what the command execution will be like.

You can also customize the `/preview` route to have its own authentication method, especially if you don't want to keep
private the content of a command. This authentication configuration follows the same rules as a
generic [listener one](/0060-authentication.md).

An example of what you can see with this plugin is:

```yaml
command: echo
args:
  - Hello Mr. Anderson
```

**SIDE EFFECTS:** when using [temporary/persistent files](/0040-local-files.md), the preview plugin will **write** to
these files in order to compute the command preview. This can lead to side effects, especially if you are using
persistent files.

## Configuration

[filename](../../pkg/plugin_preview.go ':include :type=code :fragment=config')

## Examples

This is an example on how to use the preview plugin:

> Example code at: [`/examples/config.plugin.preview.yaml`](https://github.com/cmaster11/go-to-exec/tree/main/examples/config.plugin.preview.yaml)

[filename](../../examples/config.plugin.preview.yaml ':include :type=code')