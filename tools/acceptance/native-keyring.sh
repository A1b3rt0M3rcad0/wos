#!/bin/sh
# Run the native Linux gate against an isolated, unlocked Secret Service.
set -eu
if [ "$(uname -s)" != Linux ]; then
  echo 'Linux Secret Service fixture requires a native Linux host' >&2
  exit 1
fi
if [ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ]; then
  exec dbus-run-session -- sh "$0"
fi
keyring_directory="$(mktemp -d)"
trap 'rm -rf "$keyring_directory"' EXIT
export XDG_DATA_HOME="$keyring_directory/data"
export XDG_RUNTIME_DIR="$keyring_directory/runtime"
mkdir -m 700 "$XDG_DATA_HOME" "$XDG_RUNTIME_DIR"
# A generated fixture unlock password is supplied only through stdin.
openssl rand -base64 32 | gnome-keyring-daemon --unlock --components=secrets >"$keyring_directory/daemon-state"
export WOS_TEST_NATIVE_KEYRING=1
node tools/distribution/platform.mjs record
