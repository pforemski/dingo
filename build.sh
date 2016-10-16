#!/bin/bash

[ -z "$1" ] && { echo "Usage: build.sh VERSION" >&1; exit 1; }

VERSION="$1"
DEST="$HOME/tmp/dingo-$VERSION"

###############################################

function build()
{
	TARGET="$1"

	echo "Building dingo v. $VERSION for $TARGET"
	GOOS="${TARGET%-*}" GOARCH="${TARGET##*-}" go build \
		-o $DEST/dingo-$TARGET \
		./*.go
}

###############################################

echo "Building in $DEST"
rm -fr $DEST
mkdir -p $DEST

for target in \
	darwin-386 darwin-amd64 \
	freebsd-386 freebsd-amd64 \
	linux-386 linux-amd64  \
	netbsd-386 netbsd-amd64 \
	openbsd-386 openbsd-amd64 \
	windows-386 windows-amd64; do
	build $target || exit 1
done
