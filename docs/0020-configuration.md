# Configuration

You can configure:

* A default configuration for all listeners (using the root `defaults` configuration key).
* A configuration for each listener, which will overwrite the relative entries of the default one.

Here are all available configuration entries:

[filename](../pkg/config.go ':include :type=code :fragment=config-docs')

## Configuration example

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
listeners:
  /hello:
    command: cat
```

can be remapped with `GTE_LISTENERS__HELLO_COMMAND=echo`.

Notes:

* All environment variables need to be prefixed by `GTE_`.
* Dynamic entries (where you can define any kind of keys), like the `listeners` map, **need to be defined in the initial
  config**, before they can be re-mapped using environment variables.
* The environment variable name for a config entry is created by:
    1. Join all the keys' chain with `_`: `listeners_/hello_command`
    2. Replace all non `a-z`, `A-Z`, `0-9`, `_` characters with `_`: `listeners__hello_command`
    3. Turn the whole text to upper-case: `LISTENERS__HELLO_COMMAND`
    4. Prefix with `GTE_`: `GTE_LISTENERS__HELLO_COMMAND`
