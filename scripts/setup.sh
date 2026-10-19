#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLS_DIR="$ROOT_DIR/.tools"
SDK_ROOT="$TOOLS_DIR/android-sdk"
AVD_HOME="$TOOLS_DIR/avd"
AVD_NAME="SanguoClip_API35"
ANDROID_API="35"
COMMANDLINE_TOOLS_VERSION="15859902"

WITH_EMULATOR=false
ACCEPT_LICENSES=false
SKIP_BUILD=false

usage() {
  cat <<'EOF'
Usage: ./scripts/setup.sh [options]

Options:
  --with-emulator    Install Android emulator, API 35 image, and create an AVD.
  --accept-licenses  Accept Android SDK licenses non-interactively.
  --skip-build       Install dependencies without building SanguoClip.
  -h, --help         Show this help.
EOF
}

while (($# > 0)); do
  case "$1" in
    --with-emulator)
      WITH_EMULATOR=true
      ;;
    --accept-licenses)
      ACCEPT_LICENSES=true
      ;;
    --skip-build)
      SKIP_BUILD=true
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This script supports macOS only. On Windows, run scripts/setup.ps1." >&2
  exit 1
fi

if ! command -v brew >/dev/null 2>&1; then
  cat >&2 <<'EOF'
Homebrew is required but was not found.
Install it from https://brew.sh/, then rerun this script.
EOF
  exit 1
fi

install_formula() {
  local name="$1"
  local command_name="${2:-}"
  if [[ -n "$command_name" ]] && command -v "$command_name" >/dev/null 2>&1; then
    echo "[ok] Command: $command_name"
    return
  fi
  if brew list --formula "$name" >/dev/null 2>&1; then
    echo "[ok] Homebrew formula: $name"
    return
  fi
  brew install "$name"
}

install_cask() {
  local name="$1"
  local command_name="${2:-}"
  if [[ -n "$command_name" ]] && command -v "$command_name" >/dev/null 2>&1; then
    echo "[ok] Command: $command_name"
    return
  fi
  if brew list --cask "$name" >/dev/null 2>&1; then
    echo "[ok] Homebrew cask: $name"
    return
  fi
  brew install --cask "$name"
}

mkdir -p "$TOOLS_DIR"

install_formula go go
install_formula ffmpeg ffmpeg

if "$WITH_EMULATOR"; then
  if [[ -x "$TOOLS_DIR/jdk/Contents/Home/bin/java" ]]; then
    JAVA_HOME="$TOOLS_DIR/jdk/Contents/Home"
    echo "[ok] Project-local JDK"
  else
    install_formula openjdk@21
    JAVA_HOME="$(brew --prefix openjdk@21)/libexec/openjdk.jdk/Contents/Home"
  fi

  case "$(uname -m)" in
    arm64)
      host_arch="arm64"
      image_abi="arm64-v8a"
      ;;
    x86_64)
      host_arch="x86_64"
      image_abi="x86_64"
      ;;
    *)
      echo "Unsupported macOS architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac

  if [[ ! -x "$SDK_ROOT/cmdline-tools/latest/bin/sdkmanager" ]]; then
    download_dir="$TOOLS_DIR/downloads"
    extract_dir="$TOOLS_DIR/cmdline-tools-extract"
    archive="$download_dir/android-commandlinetools.zip"
    platform_suffix="mac_$host_arch"
    tools_url="https://dl.google.com/android/repository/commandlinetools-${platform_suffix}-${COMMANDLINE_TOOLS_VERSION}_latest.zip"

    mkdir -p "$download_dir" "$SDK_ROOT/cmdline-tools"
    rm -rf "$extract_dir" "$SDK_ROOT/cmdline-tools/latest"
    echo "Downloading Android command-line tools..."
    curl --fail --location --retry 3 --output "$archive" "$tools_url"
    mkdir -p "$extract_dir"
    unzip -q "$archive" -d "$extract_dir"
    mv "$extract_dir/cmdline-tools" "$SDK_ROOT/cmdline-tools/latest"
    rm -rf "$extract_dir"
  else
    echo "[ok] Android command-line tools"
  fi

  export JAVA_HOME
  export ANDROID_HOME="$SDK_ROOT"
  export ANDROID_SDK_ROOT="$SDK_ROOT"
  export ANDROID_AVD_HOME="$AVD_HOME"
  export PATH="$JAVA_HOME/bin:$SDK_ROOT/cmdline-tools/latest/bin:$SDK_ROOT/platform-tools:$SDK_ROOT/emulator:$PATH"

  sdkmanager="$SDK_ROOT/cmdline-tools/latest/bin/sdkmanager"
  avdmanager="$SDK_ROOT/cmdline-tools/latest/bin/avdmanager"
  image_package="system-images;android-${ANDROID_API};google_apis_playstore;${image_abi}"

  if "$ACCEPT_LICENSES"; then
    set +o pipefail
    yes 2>/dev/null | "$sdkmanager" --sdk_root="$SDK_ROOT" --licenses >/dev/null
    license_status="${PIPESTATUS[1]}"
    set -o pipefail
    if [[ "$license_status" -ne 0 ]]; then
      echo "Android SDK license acceptance failed." >&2
      exit "$license_status"
    fi
  else
    echo "Review and accept the Android SDK licenses to continue."
    "$sdkmanager" --sdk_root="$SDK_ROOT" --licenses
  fi

  "$sdkmanager" --sdk_root="$SDK_ROOT" \
    "cmdline-tools;latest" \
    "platform-tools" \
    "emulator" \
    "platforms;android-${ANDROID_API}" \
    "$image_package"

  mkdir -p "$AVD_HOME"
  if ! "$avdmanager" list avd | grep -Fq "Name: $AVD_NAME"; then
    printf 'no\n' | "$avdmanager" create avd \
      --force \
      --name "$AVD_NAME" \
      --package "$image_package" \
      --device pixel_7
  else
    echo "[ok] Android virtual device: $AVD_NAME"
  fi
else
  install_cask android-platform-tools adb
fi

if [[ ! -f "$ROOT_DIR/.env" ]]; then
  cp "$ROOT_DIR/.env.example" "$ROOT_DIR/.env"
  echo "Created .env from .env.example"
fi

if ! "$SKIP_BUILD"; then
  mkdir -p "$ROOT_DIR/bin"
  (
    cd "$ROOT_DIR"
    go build -o bin/sanguoclip ./cmd/sanguoclip
  )
fi

"$ROOT_DIR/scripts/doctor.sh"

echo
echo "Setup complete."
echo "Start SanguoClip: ./scripts/run.sh"
if "$WITH_EMULATOR"; then
  echo "Start the emulator: ./scripts/start-emulator.sh"
fi
