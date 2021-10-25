# Configuration

You can configure:

* A default configuration for all listeners (using the root `defaults` configuration key).
* A configuration for each listener, which will overwrite the relative entries of the default one.

Here are all available configuration entries:

[filename](../pkg/config.go ':include :type=code :fragment=config-docs')

## Configuration example

> Example code at: [`/examples/config.simple.yaml`](https://github.com/cmaster11/go-to-exec/tree/main/examples/config.simple.yaml)

[filename](../examples/config.simple.yaml ':include :type=code')

Test with:

```bash
# A simple GET request 
curl "http://localhost:7055/hello?name=Anderson"

# A JSON POST request
curl "http://localhost:7055/hello" -d '{"name":"Anderson"}' -H 'Content-Type: application/json'
```

## Config via environment variables

Also, all configuration entries can be re-mapped via environment variables. For example:

```yaml
defaults:
  database:
    host: localhost

listeners:
  /hello:
    command: cat
```

can be remapped with

```
GTE_DEFAULTS_DATABASE_HOST=postgres
GTE_LISTENERS__HELLO_COMMAND=echo
```

Notes:

* All environment variables need to be prefixed by `GTE_`.
* Dynamic entries (where the key/value pairs belong to a dynamic map), like the `listeners` map, **need to be defined in
  the initial config**, before they can be re-mapped using environment variables.
* The environment variable name for a config entry is created by:
    1. Join all the keys' chain with `_`: `listeners_/hello_command`
    2. Replace all non `a-z`, `A-Z`, `0-9`, `_` characters with `_`: `listeners__hello_command`
    3. Turn the whole text to upper-case: `LISTENERS__HELLO_COMMAND`
    4. Prefix with `GTE_`: `GTE_LISTENERS__HELLO_COMMAND`
* When using a [multi part configuration](/0120-use-cases/multi-part-config.md), the `defaults` file will be mapped with
  the `GTE_DEFAULTS_` prefix. This means that you can use exactly the same environment variables between a `defaults`
  file and a normal configuration one.