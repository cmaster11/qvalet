FROM --platform=$BUILDPLATFORM golang:alpine AS builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o qvalet ./cmd

FROM --platform=$BUILDPLATFORM scratch

COPY --from=builder /app/qvalet /usr/bin/qvalet

ENTRYPOINT ["qvalet"]