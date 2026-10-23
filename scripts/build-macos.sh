#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v wails >/dev/null 2>&1; then
  echo "wails tidak ditemukan. Jalankan: go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0" >&2
  exit 1
fi

echo "Build frontend..."
cd desktop/frontend
npm ci
npm run build
cd "$ROOT"

echo "Build GUI..."
cd desktop
wails build -clean
cd "$ROOT"

app="$(find desktop/build/bin -maxdepth 1 -name '*.app' -print -quit)"
if [ -z "$app" ]; then
  echo "app bundle tidak ditemukan" >&2
  exit 1
fi

echo "Build helper..."
go build -trimpath -o desktop/build/bin/dl-helper ./cmd/dl-helper
cp desktop/build/bin/dl-helper "$app/Contents/MacOS/dl-helper"
chmod +x "$app/Contents/MacOS/dl-helper"

# Ad-hoc signature: valid locally, but not trusted by Gatekeeper for distribution.
codesign --force --deep --sign - "$app"

mkdir -p dist
ditto -c -k --sequesterRsrc --keepParent "$app" "dist/DevLok-macos-local.zip"

staging="$(mktemp -d)"
cp -R "$app" "$staging/DevLok.app"
ln -s /Applications "$staging/Applications"
cat > "$staging/README.txt" <<EOF
DevLok untuk macOS

Build lokal / ad-hoc signed, bukan notarized.
Prasyarat runtime Homebrew:
  brew install nginx php mysql
EOF
hdiutil create -volname DevLok -srcfolder "$staging" -ov -format UDZO "dist/DevLok-macos-local.dmg"
rm -rf "$staging"

echo "Selesai:"
ls -lh dist/DevLok-macos-local.*