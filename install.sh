#!/bin/sh
set -eu

REPO="rizqishq/cpx"
VERSION="${CPX_VERSION:-latest}"
INSTALL_DIR="${CPX_INSTALL_DIR:-$HOME/.local/bin}"

if ! command -v curl >/dev/null 2>&1; then
  printf 'Error: curl is not installed or not in PATH.\n' >&2
  exit 1
fi

if ! command -v tar >/dev/null 2>&1; then
  printf 'Error: tar is not installed or not in PATH.\n' >&2
  exit 1
fi

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *)
      printf 'Error: unsupported OS: %s\n' "$(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *)
      printf 'Error: unsupported architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
}

resolve_tag() {
  if [ "$VERSION" = "latest" ]; then
    release_json="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest")"
    tag="$(printf '%s\n' "$release_json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
    if [ -z "$tag" ]; then
      printf 'Error: could not determine the latest release tag.\n' >&2
      exit 1
    fi
    printf '%s\n' "$tag"
    return
  fi

  case "$VERSION" in
    v*) printf '%s\n' "$VERSION" ;;
    *) printf 'v%s\n' "$VERSION" ;;
  esac
}

os="$(detect_os)"
arch="$(detect_arch)"
tag="$(resolve_tag)"
asset_version="${tag#v}"
asset="cpx_${asset_version}_${os}_${arch}.tar.gz"
download_url="https://github.com/$REPO/releases/download/$tag/$asset"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

archive_path="$tmp_dir/$asset"
curl -fsSL "$download_url" -o "$archive_path"
tar -xzf "$archive_path" -C "$tmp_dir"

mkdir -p "$INSTALL_DIR"
cp "$tmp_dir/cpx" "$INSTALL_DIR/cpx"
chmod +x "$INSTALL_DIR/cpx"

printf 'Installed cpx %s to %s/cpx\n' "$tag" "$INSTALL_DIR"
printf 'Make sure %s is in your PATH.\n' "$INSTALL_DIR"
