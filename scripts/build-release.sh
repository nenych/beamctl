#!/bin/sh
# Builds the release archives for every platform into dist/ (or the directory
# given as the second argument), with a checksums.txt next to them.
# Run on macOS: the macOS builds need cgo, the others are cross-compiled.
#
#   scripts/build-release.sh v1.2.3 [dist]
set -eu

tag=${1:?usage: build-release.sh <tag> [dist]}
version=${tag#v}
dist=${2:-dist}
mkdir -p "$dist"

build() {
	goos=$1
	goarch=$2
	cgo=$3
	name=beamctl_${goos}_$goarch
	work=$dist/$name
	mkdir -p "$work"
	bin=beamctl
	[ "$goos" = windows ] && bin=beamctl.exe

	GOOS=$goos GOARCH=$goarch CGO_ENABLED=$cgo \
		go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$work/$bin" .
	cp LICENSE THIRD_PARTY_NOTICES.md "$work/"
	[ "$goos" = linux ] && cp 70-beamctl.rules "$work/"

	if [ "$goos" = windows ]; then
		(cd "$work" && zip -q "../$name.zip" ./*)
	else
		tar -czf "$dist/$name.tar.gz" -C "$work" .
	fi
	echo "built $name"
}

build darwin arm64 1
build darwin amd64 1
build linux amd64 0
build linux arm64 0
build windows amd64 0
build windows arm64 0

(cd "$dist" && shasum -a 256 ./*.tar.gz ./*.zip | sed 's|\./||' > checksums.txt)
echo "checksums:"
cat "$dist/checksums.txt"
