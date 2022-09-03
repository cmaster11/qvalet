ARG BUILDPLATFORM=amd64

FROM --platform=$BUILDPLATFORM golang:alpine AS builder
RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o qvalet ./cmd

FROM --platform=$BUILDPLATFORM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/qvalet /usr/bin/qvalet

ENTRYPOINT ["qvalet"]