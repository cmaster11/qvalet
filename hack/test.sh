#!/usr/bin/env bash
set -Eeuxmo pipefail
DIR=$(cd "$(dirname "$0")"; pwd -P)

TAG="gte-test"

(
  cd "$DIR/../.github/actions/test"
  docker build "$TAG" .
)

docker run --rm "$TAG"