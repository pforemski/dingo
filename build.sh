#!/bin/bash

[ -z "$1" ] && { echo "Usage: build.sh VERSION" >&1; exit 1; }
VERSION="$1"

###############################################

function build()
{
	TARGET="$1"

	echo "Building dingo v. $VERSION for $TARGET"
	GOOS="${TARGET%-*}" GOARCH="${TARGET##*-}" go build \
		-o release/dingo-$VERSION/dingo-$TARGET \
		./dingo.go ./gdns.go
}

###############################################

rm -fr ./release/dingo-$VERSION
mkdir -p ./release/dingo-$VERSION

for target in \
	darwin-386 darwin-amd64 \
	freebsd-386 freebsd-amd64 \
	linux-386 linux-amd64  \
	netbsd-386 netbsd-amd64 \
	openbsd-386 openbsd-amd64 \
	windows-386 windows-amd64; do
	build $target || exit 1
done
