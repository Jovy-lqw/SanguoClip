#!/usr/bin/env bash

set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK_ROOT="$ROOT_DIR/.tools/android-sdk"

if [[ -d "$SDK_ROOT" ]]; then
  export ANDROID_HOME="$SDK_ROOT"
  export ANDROID_SDK_ROOT="$SDK_ROOT"
  export ANDROID_AVD_HOME="$ROOT_DIR/.tools/avd"
  export PATH="$SDK_ROOT/platform-tools:$SDK_ROOT/emulator:$PATH"
fi

show_dependency() {
  local name="$1"
  local path
  path="$(command -v "$name" 2>/dev/null || true)"
  if [[ -n "$path" ]]; then
    printf '%-10s %s\n' "$name" "$path"
  else
    printf '%-10s missing\n' "$name"
  fi
}

show_dependency go
show_dependency adb
show_dependency ffmpeg
show_dependency ffprobe
show_dependency java
show_dependency emulator

if command -v adb >/dev/null 2>&1; then
  echo
  adb devices -l

  package_status() {
    local serial="$1"
    local package_name="$2"
    if adb -s "$serial" shell pm list packages "$package_name" 2>/dev/null |
      tr -d '\r' |
      grep -Fxq "package:$package_name"; then
      printf 'installed'
    else
      printf 'missing'
    fi
  }

  while read -r serial state _; do
    if [[ "$state" != "device" ]]; then
      continue
    fi

    game_package="com.bf.sgs.hdexp.huawei"
    deep_link_status="missing"
    if adb -s "$serial" shell \
      "cmd package query-activities --brief -a android.intent.action.VIEW -d 'yokasgs://?payload=sanguoclip-probe'" 2>/dev/null |
      tr -d '\r' |
      grep -Fq "$game_package/"; then
      deep_link_status="ready"
    fi

    echo
    echo "Huawei channel ($serial)"
    printf '  %-11s %s\n' "AppGallery" "$(package_status "$serial" "com.huawei.appmarket")"
    printf '  %-11s %s\n' "HMS Core" "$(package_status "$serial" "com.huawei.hwid")"
    printf '  %-11s %s\n' "Sanguosha" "$(package_status "$serial" "$game_package")"
    printf '  %-11s %s\n' "deep link" "$deep_link_status"
  done < <(adb devices | awk 'NR > 1 && NF >= 2 {print $1, $2}')
fi
