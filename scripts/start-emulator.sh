#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK_ROOT="$ROOT_DIR/.tools/android-sdk"
AVD_HOME="$ROOT_DIR/.tools/avd"
AVD_NAME="${SANGUOCLIP_AVD_NAME:-SanguoClip_API35}"
GPU_MODE="${SANGUOCLIP_EMULATOR_GPU:-swangle}"

legacy_avd_home="$HOME/.android/avd"
if [[ -f "$legacy_avd_home/$AVD_NAME.avd/userdata-qemu.img.qcow2" &&
  ! -f "$AVD_HOME/$AVD_NAME.avd/userdata-qemu.img.qcow2" ]]; then
  AVD_HOME="$legacy_avd_home"
fi

export ANDROID_HOME="$SDK_ROOT"
export ANDROID_SDK_ROOT="$SDK_ROOT"
export ANDROID_AVD_HOME="$AVD_HOME"
export PATH="$SDK_ROOT/platform-tools:$SDK_ROOT/emulator:$PATH"

emulator="$SDK_ROOT/emulator/emulator"
if [[ ! -x "$emulator" ]]; then
  echo "Android emulator is missing. Run ./scripts/setup.sh --with-emulator first." >&2
  exit 1
fi

if ! "$emulator" -list-avds | grep -Fxq "$AVD_NAME"; then
  echo "AVD $AVD_NAME is missing. Run ./scripts/setup.sh --with-emulator first." >&2
  exit 1
fi

avd_config="$AVD_HOME/$AVD_NAME.avd/config.ini"
if [[ -f "$avd_config" ]] && grep -q '^hw\.keyboard=no$' "$avd_config"; then
  sed -i.bak 's/^hw\.keyboard=no$/hw.keyboard=yes/' "$avd_config"
  rm -f "$avd_config.bak"
fi

exec "$emulator" \
  -avd "$AVD_NAME" \
  -no-boot-anim \
  -no-metrics \
  -crash-report-mode disabled \
  -use-keycode-forwarding \
  -gpu "$GPU_MODE"
