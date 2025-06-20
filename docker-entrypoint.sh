#!/usr/bin/env sh
set -e

if [ "$1" = 'main' ]; then
  exec su-exec app "$@"
fi

exec "$@"
