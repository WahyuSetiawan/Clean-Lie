#!/usr/bin/env bash
# Bandingkan total per item Cleanlie dengan `du -sk` (toleransi 2%).
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

echo "→ memindai (bisa beberapa detik) ..." >&2
"$BIN" --dry-run --json >"$tmp/report.json"

# Item > 100 MiB, maksimal 15 terbesar.
rows="$(jq -r '.groups[].items[] | select(.bytes > 104857600) | [.bytes, .path, .label] | @tsv' "$tmp/report.json" | sort -rn | head -15)"

printf '%-30s %12s %12s %8s\n' "ITEM" "REPORT" "DU" "DELTA"
fail=0
count=0

while IFS=$'\t' read -r bytes path label; do
  [[ -z "${path:-}" ]] && continue
  if [[ ! -e "$path" ]]; then
    printf '%-30s %12s %12s %8s\n' "${label:0:30}" "$(numfmt --to=iec "$bytes")" "-" "skip"
    continue
  fi
  kb="$(du -sk "$path" 2>/dev/null | cut -f1)" || continue
  actual=$((kb * 1024))
  delta="$(awk -v a="$bytes" -v b="$actual" 'BEGIN { d = (a - b) / b * 100; if (d < 0) d = -d; printf "%.2f", d }')"
  status="ok"
  awk -v d="$delta" 'BEGIN { exit !(d > 2.0) }' && status="BAD"
  [[ "$status" == "BAD" ]] && fail=1
  count=$((count + 1))
  printf '%-30s %12s %12s %7s%% %s\n' "${label:0:30}" "$(numfmt --to=iec "$bytes")" "$(numfmt --to=iec "$actual")" "$delta" "$status"
done <<<"$rows"

if [[ $count -eq 0 ]]; then
  echo "tidak ada item > 100 MiB untuk diverifikasi" >&2
  exit 1
fi
if [[ $fail -ne 0 ]]; then
  echo "GAGAL: ada item dengan selisih > 2%" >&2
  exit 1
fi
echo "OK: $count item dalam toleransi 2%" >&2
