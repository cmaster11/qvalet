ARG BUILDPLATFORM=amd64

FROM --platform=$BUILDPLATFORM golang:1.19-alpine3.16 AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o qvalet ./cmd

FROM --platform=$BUILDPLATFORM alpine:3.16

RUN apk update && apk --no-cache add \
    ca-certificates \
    curl \
    wget \
    jq \
    bash \
    zsh \
    && rm -vrf /var/cache/apk/*

COPY --from=builder /app/qvalet /usr/bin/qvalet

ENTRYPOINT ["qvalet"]