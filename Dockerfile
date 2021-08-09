FROM --platform=$BUILDPLATFORM golang:alpine AS builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o gotoexec ./cmd

FROM --platform=$BUILDPLATFORM scratch

COPY --from=builder /app/gotoexec /usr/bin/gotoexec

ENTRYPOINT ["gotoexec"]