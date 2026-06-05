#!/bin/sh
set -eu

OWNER="rizqishq"
REPO="cpx"
INSTALL_DIR="${CPX_INSTALL_DIR:-$HOME/.local/bin}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Error: %s is required.\n' "$1" >&2
    exit 1
  fi
}

resolve_version() {
  if [ "$#" -gt 0 ] && [ -n "$1" ]; then
    printf '%s\n' "$1"
    return
  fi

  need_cmd curl
  version="$(curl -fsSL "https://api.github.com/repos/$OWNER/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$version" ]; then
    printf 'Error: failed to resolve latest release tag.\n' >&2
    exit 1
  fi
  printf '%s\n' "$version"
}

resolve_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *)
      printf 'Error: unsupported operating system: %s\n' "$(uname -s)" >&2
      exit 1
      ;;
  esac
}

resolve_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    aarch64|arm64) printf 'arm64\n' ;;
    *)
      printf 'Error: unsupported architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
}

need_cmd curl
need_cmd tar
need_cmd mktemp

version="$(resolve_version "${1:-}")"
os="$(resolve_os)"
arch="$(resolve_arch)"
archive="cpx_${version}_${os}_${arch}.tar.gz"
url="https://github.com/$OWNER/$REPO/releases/download/$version/$archive"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

archive_path="$tmp_dir/$archive"
binary_path="$tmp_dir/cpx"

printf 'Downloading %s\n' "$url"
curl -fsSL -o "$archive_path" "$url"

tar -xzf "$archive_path" -C "$tmp_dir"

if [ ! -f "$binary_path" ]; then
  printf 'Error: cpx binary not found in downloaded archive.\n' >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
install_path="$INSTALL_DIR/cpx"
mv "$binary_path" "$install_path"
chmod +x "$install_path"

printf 'Installed cpx %s to %s\n' "$version" "$install_path"
printf 'Make sure %s is in your PATH.\n' "$INSTALL_DIR"
