#!/bin/sh
# Preserve Go/runtime dependency licenses in shipped artifacts; no new dependency.
set -eu
if [ "$#" -ne 1 ]; then
  echo 'usage: sh tools/distribution/notices.sh OUTPUT_DIRECTORY' >&2
  exit 1
fi
notice_output=$1
mkdir -p "$notice_output"
cp "$(go env GOROOT)/LICENSE" "$notice_output/Go-LICENSE.txt"
printf '%s\n' '# WOS bundled runtime notices' '' 'Go standard library: Go-LICENSE.txt' > "$notice_output/index.md"
notice_rows=$(mktemp)
trap 'rm -f "$notice_rows" "$notice_rows.sorted"' EXIT
go list -deps -f '{{if .Module}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}' ./packages/wos-api/cmd/wos > "$notice_rows"
sort -u "$notice_rows" > "$notice_rows.sorted"
while IFS='|' read -r notice_module notice_version notice_dir; do
  [ -n "$notice_version" ] || continue
  notice_name=$(printf '%s' "$notice_module@$notice_version" | tr '/' '_')
  mkdir -p "$notice_output/$notice_name"
  notice_count=0
  for notice_file in "$notice_dir"/LICENSE* "$notice_dir"/NOTICE* "$notice_dir"/COPYING*; do
    [ -f "$notice_file" ] || continue
    cp "$notice_file" "$notice_output/$notice_name/"
    notice_count=$((notice_count + 1))
  done
  if [ "$notice_count" -eq 0 ]; then
    printf 'Missing upstream license/notice for %s\n' "$notice_module" >&2
    exit 1
  fi
  printf '\n- %s@%s: %s/\n' "$notice_module" "$notice_version" "$notice_name" >> "$notice_output/index.md"
done < "$notice_rows.sorted"
