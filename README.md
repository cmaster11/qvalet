# qValet

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cmaster11)

qValet listens for HTTP requests and executes commands on demand.

[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/cmaster11/qvalet?sort=semver)](https://github.com/cmaster11/qvalet/releases)
![GitHub all releases](https://img.shields.io/github/downloads/cmaster11/qvalet/total)

## View **docs** at [https://cmaster11.github.io/qvalet/](https://cmaster11.github.io/qvalet/#/0010-getting-started)

Links:

* [Docker image](https://hub.docker.com/r/cmaster11/qvalet/tags?page=1&ordering=last_updated)
* [Some use cases](https://cmaster11.github.io/qvalet/#/0120-use-cases/)

## Feature list

* Command execution with [templatable](https://cmaster11.github.io/qvalet/#/0030-templates) fields (command, arguments, environment variables, etc…)
* Some built-in [authentication](https://cmaster11.github.io/qvalet/#/0060-authentication) methods
* [Storage](https://cmaster11.github.io/qvalet/#/0050-storage) for payloads and execution results
* [Trigger conditions](https://cmaster11.github.io/qvalet/#/0080-trigger-conditions) to evaluate if a command should be run or not
* [AWS SNS](https://cmaster11.github.io/qvalet/#/0110-plugins/awssns) support, to automatically accept subscription requests and receive AWS SNS messages
* [Scheduled tasks](https://cmaster11.github.io/qvalet/#/0110-plugins/schedule), to execute commands in the future
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