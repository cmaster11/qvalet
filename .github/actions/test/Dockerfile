FROM golang:1.16-buster

# Install tools required for testing

RUN go get -u golang.org/x/tools/cmd/goimports
RUN curl -fsSL https://deb.nodesource.com/setup_16.x | bash - && apt-get install -y nodejs

COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]