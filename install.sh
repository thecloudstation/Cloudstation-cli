#!/bin/sh
# CloudStation CLI installer.
# Usage: curl -fsSL https://raw.githubusercontent.com/thecloudstation/Cloudstation-cli/main/install.sh | sh
# Override version: CS_VERSION=v1.2.3 sh install.sh   (or pass as first arg)
set -u

REPO="thecloudstation/Cloudstation-cli"
BIN_NAME="cs"

die() { echo "cs-install: error: $*" >&2; exit 1; }

# --- fetch helpers: curl preferred, wget fallback ---
if command -v curl >/dev/null 2>&1; then
	dl() { curl -fsSL "$1" -o "$2"; }
	dl_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
	dl() { wget -qO "$2" "$1"; }
	dl_stdout() { wget -qO - "$1"; }
else
	die "need curl or wget installed to download the CLI."
fi

# --- detect OS ---
uname_s=$(uname -s)
case "$uname_s" in
	Darwin) OS=darwin ;;
	Linux)  OS=linux ;;
	MINGW*|MSYS*|CYGWIN*|Windows_NT)
		die "Windows is not supported by this script. Download cs-windows-amd64.exe from https://github.com/$REPO/releases/latest" ;;
	*) die "unsupported OS '$uname_s'. See https://github.com/$REPO/releases/latest" ;;
esac

# --- detect arch ---
uname_m=$(uname -m)
case "$uname_m" in
	x86_64|amd64) ARCH=amd64 ;;
	arm64|aarch64) ARCH=arm64 ;;
	*) die "unsupported architecture '$uname_m'. See https://github.com/$REPO/releases/latest" ;;
esac

# --- resolve version (arg > env > latest tag from GitHub API) ---
TAG="${1:-${CS_VERSION:-}}"
if [ -z "$TAG" ]; then
	TAG=$(dl_stdout "https://api.github.com/repos/$REPO/releases/latest" \
		| grep '"tag_name"' | head -n1 | sed -E 's/.*"tag_name"[^"]*"([^"]+)".*/\1/')
	[ -n "$TAG" ] || die "could not resolve the latest release tag from the GitHub API."
fi
# normalize to a leading 'v'
case "$TAG" in v*) : ;; *) TAG="v$TAG" ;; esac

ASSET="cs-${OS}-${ARCH}"
BASE="https://github.com/$REPO/releases/download/$TAG"

# --- sha256 tool ---
if command -v shasum >/dev/null 2>&1; then
	sha256() { shasum -a 256 "$1" | awk '{print $1}'; }
elif command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | awk '{print $1}'; }
else
	die "need shasum or sha256sum to verify the download."
fi

# --- download to a temp dir ---
TMP=$(mktemp -d) || die "could not create a temp directory."
trap 'rm -rf "$TMP"' EXIT INT TERM

echo "cs-install: downloading $ASSET ($TAG)..." >&2
dl "$BASE/$ASSET" "$TMP/$ASSET" || die "download failed: $BASE/$ASSET"
dl "$BASE/checksums.txt" "$TMP/checksums.txt" || die "download failed: $BASE/checksums.txt"

# --- verify checksum ---
want=$(grep " $ASSET\$" "$TMP/checksums.txt" | awk '{print $1}' | head -n1)
[ -n "$want" ] || die "no checksum for $ASSET in checksums.txt."
got=$(sha256 "$TMP/$ASSET")
[ "$want" = "$got" ] || die "checksum mismatch for $ASSET (want $want, got $got)."

# --- pick install dir ---
if [ -n "${CS_INSTALL_DIR:-}" ]; then
	DEST="$CS_INSTALL_DIR"
elif [ -w /usr/local/bin ] 2>/dev/null; then
	DEST=/usr/local/bin
else
	DEST="$HOME/.local/bin"
fi
mkdir -p "$DEST" || die "could not create install dir '$DEST'."

# --- install binary + symlink ---
install -m 0755 "$TMP/$ASSET" "$DEST/$BIN_NAME" 2>/dev/null \
	|| { cp "$TMP/$ASSET" "$DEST/$BIN_NAME" && chmod 0755 "$DEST/$BIN_NAME"; } \
	|| die "could not write '$DEST/$BIN_NAME' (try CS_INSTALL_DIR=... or run with sudo)."
ln -sf "$BIN_NAME" "$DEST/cloudstation" || die "could not create 'cloudstation' symlink in '$DEST'."

# --- PATH hint ---
case ":$PATH:" in
	*":$DEST:"*) ;;
	*)
		echo "cs-install: '$DEST' is not on your PATH. Add it:" >&2
		sh_name=$(basename "${SHELL:-sh}")
		case "$sh_name" in
			fish) echo "  fish_add_path $DEST" >&2 ;;
			*)    echo "  export PATH=\"$DEST:\$PATH\"" >&2 ;;
		esac
		;;
esac

# --- verify + success message ---
"$DEST/$BIN_NAME" --version >/dev/null 2>&1 || die "installed binary failed to run '$BIN_NAME --version'."
ver=$("$DEST/$BIN_NAME" --version 2>/dev/null | head -n1)
echo "cs-install: installed $ver to $DEST/$BIN_NAME — run 'cs update' to upgrade later, and 'cs agent-guide' to get started."
