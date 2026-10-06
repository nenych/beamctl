#!/bin/sh
# Points the Homebrew formula in nenych/homebrew-tap at a released tag.
# Needs HOMEBREW_TAP_TOKEN, a token that may push to that repository.
#
#   scripts/update-tap.sh v1.2.3
set -eu

tag=${1:?usage: update-tap.sh <tag>}
tarball=https://github.com/nenych/beamctl/archive/refs/tags/$tag.tar.gz
sha=$(curl -fsSL "$tarball" | shasum -a 256 | cut -d' ' -f1)

work=$(mktemp -d)
git clone -q "https://x-access-token:$HOMEBREW_TAP_TOKEN@github.com/nenych/homebrew-tap.git" "$work"
formula=$work/Formula/beamctl.rb
sed -i.bak -e "s|^  url \".*\"|  url \"$tarball\"|" -e "s|^  sha256 \".*\"|  sha256 \"$sha\"|" "$formula"
rm "$formula.bak"

cd "$work"
git -c user.name="github-actions[bot]" -c user.email="41898282+github-actions[bot]@users.noreply.github.com" \
	commit -q -am "beamctl $tag" && git push -q
echo "formula now points at $tag ($sha)"
