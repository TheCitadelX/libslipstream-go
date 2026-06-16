#!/usr/bin/env sh
set -eu

OUTPUT="${OUTPUT:-build/ios/Libslipstream.framework}"
TARGET="${TARGET:-ios}"
PREFIX="${PREFIX:-Slipstream}"

if [ "$(uname -s)" != "Darwin" ]; then
  echo "iOS bindings must be built on macOS with Xcode installed." >&2
  exit 1
fi

if ! command -v gomobile >/dev/null 2>&1; then
  echo "gomobile is not installed. Run: go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT")"

gomobile bind \
  -target="$TARGET" \
  -prefix="$PREFIX" \
  -trimpath \
  -o "$OUTPUT" \
  ./mobile
