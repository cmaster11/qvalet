# Getting started

To get a very simple listener running on any linux/Mac device, you can use:

```bash
# https://raw.githubusercontent.com/cmaster11/go-to-exec/main/hack/run.sh
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
gotoexec --config examples/config.simple.yaml
```

* You can download `go-to-exec` binaries from the [releases](https://github.com/cmaster11/go-to-exec/releases)
page. [![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/cmaster11/go-to-exec?sort=semver)](https://github.com/cmaster11/go-to-exec/releases)
* The docker image `cmaster11/go-to-exec` is served on [Docker Hub](https://hub.docker.com/r/cmaster11/go-to-exec).
To run the docker image on e.g. a local Windows machine:

```bash
docker run -i -t -v "C:/path/to/config.yaml:/mnt/config.yaml" --rm cmaster11/go-to-exec --config /mnt/config.yaml 
```

* You can see a simple Kubernetes deployment example in our [use cases](/0120-use-cases/kubernetes.md).