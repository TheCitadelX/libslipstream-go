#!/usr/bin/env sh
set -eu

OUTPUT="${OUTPUT:-build/android/libslipstream.aar}"
TARGET="${TARGET:-android}"
JAVAPKG="${JAVAPKG:-com.thecitadelx.slipstream}"

if ! command -v gomobile >/dev/null 2>&1; then
  echo "gomobile is not installed. Run: go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT")"

gomobile bind \
  -target="$TARGET" \
  -javapkg="$JAVAPKG" \
  -trimpath \
  -o "$OUTPUT" \
  ./mobile
