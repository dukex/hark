#!/bin/sh
set -eu

url=${HARK_DOWNLOAD_URL:-https://github.com/dukex/hark/releases/latest/download}
dir=${HARK_INSTALL_DIR:-$HOME/.local/bin}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case $os in
linux | darwin) ;;
*) echo "hark: unsupported OS $os" >&2 && exit 1 ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) echo "hark: unsupported architecture $(uname -m)" >&2 && exit 1 ;;
esac

asset=hark_${os}_${arch}.tar.gz
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "$url/$asset" -o "$tmp/$asset"
curl -fsSL "$url/checksums.txt" -o "$tmp/checksums.txt"
if command -v sha256sum >/dev/null; then sha="sha256sum"; else sha="shasum -a 256"; fi
(cd "$tmp" && grep "  $asset\$" checksums.txt | $sha -c - >/dev/null)

tar -xzf "$tmp/$asset" -C "$tmp" hark
mkdir -p "$dir"
mv "$tmp/hark" "$dir/hark"
chmod 755 "$dir/hark"
echo "hark installed to $dir/hark"
case ":$PATH:" in
*":$dir:"*) ;;
*) echo "add $dir to your PATH to run hark" ;;
esac
