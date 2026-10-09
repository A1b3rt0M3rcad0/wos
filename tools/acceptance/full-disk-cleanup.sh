#!/usr/bin/env bash
set -euo pipefail
# Bounded, disconnected, disposable tmpfs; never fill the workspace filesystem.
binary="$(mktemp /tmp/wos-full-disk.XXXXXX)"
trap 'rm -f "$binary"' EXIT
go test -c -o "$binary" ./packages/wos-cli
chmod 755 "$binary"
docker run --rm --network none --read-only --cap-drop ALL \
  --security-opt no-new-privileges --user 10001:10001 \
  --tmpfs /tmp:rw,size=64m,mode=1777 --tmpfs /fixture:rw,size=1m,mode=1777 \
  --mount "type=bind,source=$binary,target=/wos-tests,readonly" \
  -e WOS_TEST_FULL_DISK_DIRECTORY=/fixture postgres:18.6 \
  /wos-tests -test.v -test.run '^TestReceiptCleanupRecoversRealFullDisk$'
