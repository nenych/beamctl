#!/bin/sh
# Installs the latest beamctl release on macOS or Linux into ~/.local/bin
# (or $BEAMCTL_INSTALL_DIR), after checking the archive's SHA-256 against the
# release's checksums.txt.
#
#   curl -fsSL https://raw.githubusercontent.com/nenych/beamctl/main/install.sh | sh
set -eu

repo=nenych/beamctl
dir=${BEAMCTL_INSTALL_DIR:-$HOME/.local/bin}

case $(uname -s) in
Darwin) os=darwin ;;
Linux) os=linux ;;
*)
	echo "This script supports macOS and Linux. On Windows, download a release archive from" >&2
	echo "https://github.com/$repo/releases or run: go install github.com/$repo@latest" >&2
	exit 1
	;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*)
	echo "Unsupported CPU architecture: $(uname -m)" >&2
	exit 1
	;;
esac

asset=beamctl_${os}_$arch.tar.gz
base=https://github.com/$repo/releases/latest/download
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $asset..."
curl -fsSL -o "$tmp/$asset" "$base/$asset"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"

cd "$tmp"
if command -v sha256sum >/dev/null; then
	grep "  $asset\$" checksums.txt | sha256sum -c - >/dev/null
else
	grep "  $asset\$" checksums.txt | shasum -a 256 -c - >/dev/null
fi
tar -xzf "$asset"

mkdir -p "$dir"
install -m 755 beamctl "$dir/beamctl"
echo "Installed $("$dir/beamctl" version) to $dir/beamctl"

case ":$PATH:" in
*":$dir:"*) ;;
*) echo "Note: $dir is not in your PATH." ;;
esac

if [ "$os" = linux ]; then
	cat <<EOF

On Linux, non-root users need a udev rule before they can talk to the light.
Install it once, then reconnect the light:

  sudo curl -fsSL -o /etc/udev/rules.d/70-beamctl.rules https://raw.githubusercontent.com/$repo/main/70-beamctl.rules
  sudo udevadm control --reload && sudo udevadm trigger

See https://github.com/$repo#linux-access-to-the-light for use over SSH or from services.
EOF
fi
