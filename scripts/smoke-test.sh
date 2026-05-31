#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

cd "$ROOT"
go build -o "$TMP/kk" ./cmd/kk

mkdir "$TMP/repo"
cd "$TMP/repo"
"$TMP/kk" init
"$TMP/kk" config user.email test@example.com
"$TMP/kk" config user.name "KK Test"

branch="$($TMP/kk branch --show-current)"
if [[ "$branch" != "main" ]]; then
  echo "expected main branch, got $branch" >&2
  exit 1
fi

"$TMP/kk" track "*.bin"
echo "hello" > note.txt
head -c 1024 /dev/urandom > asset.bin
"$TMP/kk" add note.txt asset.bin

grep -q "version kk-lfs-1.0.0" asset.bin
"$TMP/kk" commit -m "initial"
"$TMP/kk" status --json >/dev/null
"$TMP/kk" fsck --json >/dev/null
"$TMP/kk" pull-file asset.bin
if grep -q "version kk-lfs-1.0.0" asset.bin; then
  echo "asset.bin was not materialized" >&2
  exit 1
fi
"$TMP/kk" dematerialize asset.bin
grep -q "version kk-lfs-1.0.0" asset.bin
OID="$(grep '^oid sha256:' asset.bin | sed 's/oid sha256://')"

"$TMP/kk" checkout -b branch-b >/dev/null
"$TMP/kk" checkout main >/dev/null
"$TMP/kk" rm asset.bin >/dev/null
"$TMP/kk" commit -m "delete asset on main" >/dev/null
"$TMP/kk" objects refs "$OID" --json | grep -q "$OID"
"$TMP/kk" objects prune --dry-run --json > "$TMP/prune.json"
if grep -q "$OID" "$TMP/prune.json"; then
  echo "live object was incorrectly marked for prune" >&2
  exit 1
fi

mkdir -p "$TMP/remote"
"$TMP/kk" remote add local studio-nas --path "$TMP/remote" --priority 10 --pull true --push true
"$TMP/kk" remote list --json >/dev/null
"$TMP/kk" remote check studio-nas --json >/dev/null

echo "smoke test passed"
