#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

OUT="${1:-dist/licenses}"
mkdir -p "$OUT"

if ! command -v go-licenses >/dev/null 2>&1; then
  echo "go-licenses belum ada, memasang..."
  go install github.com/google/go-licenses@latest
  export PATH="$(go env GOPATH)/bin:$PATH"
fi

echo "Audit lisensi Go -> $OUT/go-licenses.csv"
go-licenses csv ./... --ignore devlok > "$OUT/go-licenses.csv"

echo "Audit lisensi frontend -> $OUT/npm-licenses.txt"
(cd desktop/frontend && npx --yes license-checker --summary) > "$OUT/npm-licenses.txt"

risky='GPL|AGPL|LGPL|SSPL|Commons Clause'
if grep -Ei "$risky" "$OUT/go-licenses.csv" "$OUT/npm-licenses.txt"; then
  echo "PERINGATAN: lisensi berisiko ditemukan (lihat di atas)." >&2
  exit 2
fi

echo "Selesai. Tidak ada lisensi berisiko terdeteksi."
