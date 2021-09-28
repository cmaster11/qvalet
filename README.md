# go-to-exec

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cmaster11)

`go-to-exec` listens for HTTP requests and executes commands on demand.

## View **docs** at [https://cmaster11.github.io/go-to-exec/](https://cmaster11.github.io/go-to-exec/)

Links:

* [Docker image](https://hub.docker.com/r/cmaster11/go-to-exec/tags?page=1&ordering=last_updated)
* [Some use cases](https://cmaster11.github.io/go-to-exec/#/0120-use-cases)

## Feature list

* Command execution with [templatable](https://cmaster11.github.io/go-to-exec/#/0090-template-functions) fields (command, arguments, environment variables, etc…)
* Some built-in [authentication](https://cmaster11.github.io/go-to-exec/#/0060-authentication) methods
* [Storage](https://cmaster11.github.io/go-to-exec/#/0050-storage) for payloads and execution results
* [Trigger conditions](https://cmaster11.github.io/go-to-exec/#/0080-trigger-conditions) to evaluate if a command should be run or not
* [AWS SNS](https://cmaster11.github.io/go-to-exec/#/0110-plugins/awssns) support, to automatically accept subscription requests and receive AWS SNS messages
* And more!

## Example

```yaml
listeners:

  /hello:

    # Returns the output of the command in the response
    return: output

    # Command to run, and list of arguments
    command: bash
    args:
      - -c
      # You can use templates to customize commands/arguments/env vars
      - |
        echo "Hello {{ .name }}"
```

Tested with:

```bash
curl "http://localhost:7055/hello?name=Mr.%20Anderson"
```

Will return:

```bash
{"output":"Hello Mr. Anderson\n"}
```