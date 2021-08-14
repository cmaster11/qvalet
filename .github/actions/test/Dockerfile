FROM golang:1.16-buster

# Install tools required for testing

RUN go get -u golang.org/x/tools/cmd/goimports
RUN curl -fsSL https://deb.nodesource.com/setup_16.x | bash - && apt-get install -y nodejs

RUN apt-get update && apt-get install -y \
  jq \
  && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]