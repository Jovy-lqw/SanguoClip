#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK_ROOT="$ROOT_DIR/.tools/android-sdk"

if [[ -d "$SDK_ROOT" ]]; then
  export ANDROID_HOME="$SDK_ROOT"
  export ANDROID_SDK_ROOT="$SDK_ROOT"
  export ANDROID_AVD_HOME="$ROOT_DIR/.tools/avd"
  export PATH="$SDK_ROOT/platform-tools:$SDK_ROOT/emulator:$PATH"
fi

if [[ -f "$ROOT_DIR/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
  set +a
fi

listen="${SANGUOCLIP_ADDR:-127.0.0.1:8787}"
data_dir="${SANGUOCLIP_DATA_DIR:-$ROOT_DIR/data}"

while (($# > 0)); do
  case "$1" in
    --listen)
      listen="$2"
      shift
      ;;
    --data-dir)
      data_dir="$2"
      shift
      ;;
    -h | --help)
      echo "Usage: ./scripts/run.sh [--listen HOST:PORT] [--data-dir PATH]"
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 2
      ;;
  esac
  shift
done

cd "$ROOT_DIR"
if [[ -x "$ROOT_DIR/bin/sanguoclip" ]]; then
  exec "$ROOT_DIR/bin/sanguoclip" -addr "$listen" -data-dir "$data_dir"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "SanguoClip is not built and Go is missing. Run ./scripts/setup.sh first." >&2
  exit 1
fi

exec go run ./cmd/sanguoclip -addr "$listen" -data-dir "$data_dir"
