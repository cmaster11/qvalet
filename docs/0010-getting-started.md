# Getting started

> qValet listens for HTTP requests and executes commands on demand.

To get a very simple listener running on any linux/Mac device, you can use:

```bash
# https://raw.githubusercontent.com/cmaster11/qvalet/main/hack/run.sh
$(wget -qO - "https://git.io/JRNmT" | bash /dev/stdin -t) -c - << EOF
debug: true
listeners:
  /hello:
    command: echo
    args:
      - Hello {{ .name }}
EOF
```

And, on a separate terminal, run:

```bash
curl "http://localhost:7055/hello" -d name=Rose
```

Or, if you already have a config file you want to use:

```bash
$(wget -O - "https://git.io/JRNmT" | bash /dev/stdin -t) -c "$CONFIG_FILE"
```

## How to run

Run with:

```bash
# Go version
go run ./cmd --config examples/config.simple.yaml

# Compiled version
qvalet --config examples/config.simple.yaml
```

* You can download qValet binaries from the [releases](https://github.com/cmaster11/qvalet/releases)
page. [![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/cmaster11/qvalet?sort=semver)](https://github.com/cmaster11/qvalet/releases)
* The docker image `cmaster11/qvalet` is served on [Docker Hub](https://hub.docker.com/r/cmaster11/qvalet).
To run the docker image on e.g. a local Windows machine:

```bash
docker run -i -t -v "C:/path/to/config.yaml:/mnt/config.yaml" --rm cmaster11/qvalet --config /mnt/config.yaml 
```

* You can see a simple Kubernetes deployment example in our [use cases](/0120-use-cases/kubernetes.md).