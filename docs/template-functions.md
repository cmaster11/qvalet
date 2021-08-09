# Template functions

`go-to-exec` lets you use all function from the [sprig](https://github.com/Masterminds/sprig) library, and the following additional functions:

| Function name | Args | Output | Example |
|---|---|---|---|
| `fileReadToString` | `path` | Reads the file at `path` and returns its content as string | `fileReadToString "hello.txt"` |
| `yamlDecode` | `text` | Decodes `text` into a usable map (works only with YAML maps!) | `(yamlDecode "name: Mr. Anderson").name` |
| `yamlToJson` | `text` | Decodes `text` as YAML and re-encodes it as JSON (works only with YAML maps!) | `yamlToJson "name: Mr. Anderson"` |