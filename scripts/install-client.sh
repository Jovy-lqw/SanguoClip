#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK_ROOT="$ROOT_DIR/.tools/android-sdk"
OFFICIAL_URL="https://m.sanguosha.com/"

if [[ -d "$SDK_ROOT" ]]; then
  export PATH="$SDK_ROOT/platform-tools:$PATH"
fi

usage() {
  cat <<EOF
Usage: ./scripts/install-client.sh /path/to/client.apk [device-serial] [package-name]

Official client:
  $OFFICIAL_URL

Channel clients should be downloaded from their channel's official app store.
The package name is normally detected after installation. Pass it explicitly
only when updating one of multiple installed clients.
EOF
}

if (($# == 0)); then
  if command -v open >/dev/null 2>&1; then
    open "$OFFICIAL_URL"
  fi
  usage
  exit 2
fi

apk_path="$1"
serial="${2:-}"
package_name="${3:-}"

if [[ ! -f "$apk_path" ]]; then
  echo "APK not found: $apk_path" >&2
  exit 1
fi

if ! command -v adb >/dev/null 2>&1; then
  echo "ADB is missing. Run ./scripts/setup.sh first." >&2
  exit 1
fi

if ! unzip -tq "$apk_path" >/dev/null; then
  echo "The selected file is not a valid APK archive: $apk_path" >&2
  exit 1
fi

if [[ -z "$serial" ]]; then
  devices="$(adb devices | awk 'NR > 1 && $2 == "device" {print $1}')"
  device_count="$(printf '%s\n' "$devices" | awk 'NF {count++} END {print count + 0}')"
  if [[ "$device_count" -ne 1 ]]; then
    echo "Expected exactly one online Android device; found $device_count." >&2
    adb devices -l >&2
    echo "Pass a device serial as the second argument." >&2
    exit 1
  fi
  serial="$devices"
fi

list_game_packages() {
  adb -s "$serial" shell \
    "cmd package query-activities --brief -a android.intent.action.VIEW -d 'yokasgs://?payload=sanguoclip-probe'" 2>/dev/null |
    tr -d '\r' |
    awk -F/ '/^[[:space:]]*[A-Za-z0-9_.]+\\// {gsub(/^[[:space:]]+/, "", $1); print $1}' |
    sort -u
}

list_third_party_packages() {
  adb -s "$serial" shell pm list packages -3 2>/dev/null |
    tr -d '\r' |
    sed 's/^package://' |
    sort -u
}

before_game_packages="$(list_game_packages)"
before_packages="$(list_third_party_packages)"
adb -s "$serial" install -r "$apk_path"
after_game_packages="$(list_game_packages)"
after_packages="$(list_third_party_packages)"

if [[ -z "$package_name" ]]; then
  new_game_packages="$(comm -13 <(printf '%s\n' "$before_game_packages") <(printf '%s\n' "$after_game_packages"))"
  if [[ "$(printf '%s\n' "$new_game_packages" | awk 'NF {count++} END {print count + 0}')" -eq 1 ]]; then
    package_name="$(printf '%s\n' "$new_game_packages" | awk 'NF {print; exit}')"
  elif [[ "$(printf '%s\n' "$after_game_packages" | awk 'NF {count++} END {print count + 0}')" -eq 1 ]]; then
    package_name="$(printf '%s\n' "$after_game_packages" | awk 'NF {print; exit}')"
  else
    new_packages="$(comm -13 <(printf '%s\n' "$before_packages") <(printf '%s\n' "$after_packages"))"
    if [[ "$(printf '%s\n' "$new_packages" | awk 'NF {count++} END {print count + 0}')" -eq 1 ]]; then
      package_name="$(printf '%s\n' "$new_packages" | awk 'NF {print; exit}')"
    fi
  fi
fi

if [[ -z "$package_name" ]]; then
  echo "The APK was installed, but its package could not be selected automatically." >&2
  echo "Installed Sanguosha clients:" >&2
  printf '%s\n' "$after_game_packages" >&2
  echo "Run this script again with the package name as the third argument." >&2
  exit 1
fi

component="$(adb -s "$serial" shell cmd package resolve-activity --brief \
  -a android.intent.action.MAIN \
  -c android.intent.category.LAUNCHER \
  "$package_name" |
  tr -d '\r' |
  awk '/^[A-Za-z0-9_.]+\\// {component=$0} END {print component}')"
if [[ -z "$component" ]]; then
  echo "Installed $package_name, but no launcher activity was found." >&2
  exit 1
fi

adb -s "$serial" shell am start -W -n "$component"

echo
echo "The client is installed and open: $package_name"
echo "Review its agreements and sign in directly in the Android device."
