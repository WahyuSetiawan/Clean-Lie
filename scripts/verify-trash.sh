#!/usr/bin/env bash
# Verifikasi alur trash/undo CLI tanpa menyentuh trash asli.
# Pemakaian: scripts/verify-trash.sh [binary]
set -euo pipefail
cd "$(dirname "$0")/.."

BIN="${1:-./bin/cleanlie}"
if [[ ! -x "$BIN" ]]; then
  echo "binary tidak ditemukan: $BIN (jalankan: make build)" >&2
  exit 1
fi
if ! command -v jq >/dev/null; then
  echo "butuh jq" >&2
  exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
export HOME="$tmp/home"
export XDG_DATA_HOME="$tmp/data"
export XDG_STATE_HOME="$tmp/state"
export CLEANLIE_NO_GIO=1
mkdir -p "$HOME/orig" "$XDG_DATA_HOME/Trash/files" "$XDG_DATA_HOME/Trash/info" "$XDG_STATE_HOME/cleanlie"

root="$XDG_DATA_HOME/Trash"
seed() {
  local name="$1" size="$2"
  head -c "$size" /dev/zero >"$root/files/$name"
  printf '[Trash Info]\nPath=%s/orig/%s\nDeletionDate=2026-11-25T10:00:00\n' "$HOME" "$name" >"$root/info/$name.trashinfo"
}
seed a 1024
seed b 2048
seed c 3072
seed d 4096

# Satu entri audit untuk menguji --undo (batch uji, bukan lewat GUI).
cat >"$XDG_STATE_HOME/cleanlie/history.jsonl" <<EOF
{"time":"2026-11-25T10:00:00Z","mode":"trash","id":"audit-b","path":"$HOME/orig/b","bytesBefore":2048,"bytesAfter":0,"freed":0,"status":"removed","trashPath":"$root/files/b","batch":"verify-batch","action":"clean"}
EOF

json="$tmp/list.json"
"$BIN" --trash-list --json >"$json"
count="$(jq '.entries | length' "$json")"
[[ "$count" == "4" ]] || { echo "GAGAL: jumlah trash $count != 4" >&2; exit 1; }
id() { jq -r --arg n "$1" '.entries[] | select(.name == $n) | .id' "$json"; }

id_a="$(id a)"
id_c="$(id c)"
[[ -n "$id_a" && -n "$id_c" ]] || { echo "GAGAL: ID fixture tidak ditemukan" >&2; exit 1; }

"$BIN" --trash-restore "$id_a" --yes --json >"$tmp/restore.json"
[[ "$(jq -r '.restored' "$tmp/restore.json")" == "1" ]] || { echo "GAGAL: restore a" >&2; exit 1; }
[[ -f "$HOME/orig/a" ]] || { echo "GAGAL: file a tidak kembali" >&2; exit 1; }

"$BIN" --undo verify-batch --yes --json >"$tmp/undo.json"
[[ "$(jq -r '.restored' "$tmp/undo.json")" == "1" ]] || { echo "GAGAL: undo b" >&2; exit 1; }
[[ -f "$HOME/orig/b" ]] || { echo "GAGAL: file b tidak kembali lewat undo" >&2; exit 1; }

"$BIN" --trash-delete "$id_c" --yes --json >"$tmp/delete.json"
[[ "$(jq -r '.deleted' "$tmp/delete.json")" == "1" ]] || { echo "GAGAL: delete c" >&2; exit 1; }
[[ ! -e "$root/files/c" ]] || { echo "GAGAL: file c masih ada" >&2; exit 1; }

"$BIN" --trash-empty --yes --json >"$tmp/empty.json"
[[ "$(jq -r '.deleted' "$tmp/empty.json")" -ge 1 ]] || { echo "GAGAL: empty sisa" >&2; exit 1; }
[[ ! -e "$root/files/d" ]] || { echo "GAGAL: file d masih ada" >&2; exit 1; }

echo "OK: trash list/restore/undo/delete/empty (4 fixture, tanpa trash asli)" >&2
