#!/usr/bin/env bash
set -Eeumo pipefail

REPO=cmaster11/go-to-exec

# https://gist.github.com/lukechilds/a83e1d7127b78fef38c2914c4ececc3c
get_latest_release() {
  curl --silent "https://api.github.com/repos/$REPO/releases/latest" | \
    grep '"tag_name":' | \
    sed -E 's/.*"([^"]+)".*/\1/'
}

RELEASE=$(get_latest_release)

OS=linux
ARCH=amd64
if [ "$(uname)" == "Darwin" ]; then
  OS=darwin
fi

case $(uname -m) in
    i386 | i686)   ARCH="386" ;;
    x86_64) ARCH="amd64" ;;
    arm)    dpkg --print-architecture | grep -q "arm64" && ARCH="arm64" || ARCH="arm" ;;
esac

# Download the right executable
URL="https://github.com/$REPO/releases/download/$RELEASE/gotoexec-$OS-$ARCH"

GTE=$(mktemp)
wget -q -O "$GTE" "$URL"
chmod +x "$GTE"

# Run go-to-exec
exec "$GTE" --config -