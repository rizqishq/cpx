#!/bin/sh
set -eu

REPO="github.com/rizqishq/cpx"

if ! command -v go >/dev/null 2>&1; then
  printf 'Error: go is not installed or not in PATH.\n' >&2
  exit 1
fi

go install "${REPO}@latest"

if [ -n "${GOBIN:-}" ]; then
  bin_dir="$GOBIN"
else
  bin_dir="$(go env GOPATH)/bin"
fi

printf 'Installed cpx to %s/cpx\n' "$bin_dir"
printf 'Make sure %s is in your PATH.\n' "$bin_dir"
