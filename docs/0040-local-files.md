# Local files

You can also define local files to be written and used at runtime by creating entries in the `files` list.

Example:

```yaml
files:
  tmp1: Hello {{ .name }}
  /opt/tmp2: This is a file in an absolute route!

  my_dump: |
    Here is the dump of the request:

    {{ dump . }}
```

If the key is a relative route, it will be relative to an always-changing temporary location provided by the system (
e.g. `/tmp/qv-1234`), and the files will be **temporary**, so they will be deleted after the listener execution.

If, instead, the path is an absolute one, the files will be **persistent**, and will NOT be deleted after each call.
But, **beware** on using persistent files unless you know what you are doing! If there are two concurrent writes to the
same persistent file, you may end up having errors and/or broken/corrupted data! Use the absolute path approach ONLY if
you know what you are doing.

All files' paths will be accessible also as environment variables (with the `QV_FILES_` prefix) and template vars (
under the `(qv).files` map).

```
/tmp/key1 -> QV_FILES__tmp_key1, {{ (qv).files._tmp_key1 }}
key2 -> QV_FILES_key2, {{ (qv).files.tmp_key2 }}
```

NOTE: in environment variables and in the templates map's keys, all `\W` characters (NOT `a-z`, `A-Z`, `0-9`, `_`) will
be replaced with `_`.

## Examples

This is an example on how to use temporary files:

> Example code at: [`/examples/config.files.temporary.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.files.temporary.yaml)

[filename](../examples/config.files.temporary.yaml ':include :type=code')

And this is an example for persistent files:

> Example code at: [`/examples/config.files.persistent.yaml`](https://github.com/cmaster11/qvalet/tree/main/examples/config.files.persistent.yaml)

[filename](../examples/config.files.persistent.yaml ':include :type=code')