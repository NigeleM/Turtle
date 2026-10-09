#!/bin/sh
# Builds Turtle's Linux downloads, for Intel/AMD (amd64) and ARM (arm64):
#
#   turtle-linux-<arch>.deb      Debian and Ubuntu: turtle in /usr/bin
#   turtle-linux-<arch>.tar.gz   any Linux: turtle, LICENSE, NOTICE
#
#   installer/linux/build.sh <version> [output folder]
#
# Needs Go and dpkg-deb (Debian and Ubuntu have it).
set -eu
version=${1:?usage: build.sh <version> [output folder]}
version=${version#v}
repo=$(cd "$(dirname "$0")/../.." && pwd)
out=$(mkdir -p "${2:-$repo/release}" && cd "${2:-$repo/release}" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$repo"

for arch in amd64 arm64; do
	bin="$work/turtle-$arch"
	CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -ldflags "-s -w -X main.version=v$version" -o "$bin" ./cmd/turtle

	# The .tar.gz: one folder, to unpack anywhere.
	dir="$work/tar-$arch/turtle-$version-linux-$arch"
	mkdir -p "$dir"
	cp "$bin" "$dir/turtle"
	cp LICENSE NOTICE "$dir/"
	tar -C "$work/tar-$arch" -czf "$out/turtle-linux-$arch.tar.gz" "turtle-$version-linux-$arch"

	# The .deb.
	root="$work/deb-$arch"
	mkdir -p "$root/DEBIAN" "$root/usr/bin" "$root/usr/share/doc/turtle"
	install -m 755 "$bin" "$root/usr/bin/turtle"
	install -m 644 LICENSE "$root/usr/share/doc/turtle/copyright"
	install -m 644 NOTICE "$root/usr/share/doc/turtle/NOTICE"
	cat >"$root/DEBIAN/control" <<CONTROL
Package: turtle
Version: $version
Architecture: $arch
Maintainer: Nigele McCoy <https://github.com/NigeleM/Turtle>
Section: devel
Priority: optional
Homepage: https://github.com/NigeleM/Turtle
Description: The Turtle programming language
 A simple language that reads like sentences, with a standard library
 for data, SQL, the web and more, a prompt, a formatter and a debugger.
CONTROL
	dpkg-deb --root-owner-group --build "$root" "$out/turtle-linux-$arch.deb" >/dev/null
	echo "built $out/turtle-linux-$arch.deb and .tar.gz"
done
