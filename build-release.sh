#!/bin/bash

name='dingo'

MD5='md5sum'
if [[ "$(uname)" == 'Darwin' ]]; then
	MD5='md5'
fi

UPX=false
if hash upx 2>/dev/null; then
	UPX=true
fi

VERSION=`date -u +%Y%m%d`
LDFLAGS="-X main.version=$VERSION -s -w -linkmode external -extldflags -static"
GCFLAGS=""

# X86
OSES=(windows linux darwin freebsd)
ARCHS=(amd64 386)
rm -rf ./release
mkdir -p ./release
for os in ${OSES[@]}; do
	for arch in ${ARCHS[@]}; do
		suffix=""
		if [ "$os" == "windows" ]; then
			suffix=".exe"
			LDFLAGS="-X main.version=$VERSION -s -w"
		fi
		env CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o ./release/${name}_${os}_${arch}${suffix} .
		if $UPX; then upx -9 ./release/${name}_${os}_${arch}${suffix}; fi
		tar -C ./release -zcf ./release/${name}_${os}-${arch}-$VERSION.tar.gz ./${name}_${os}_${arch}${suffix}
		$MD5 ./release/${name}_${os}-${arch}-$VERSION.tar.gz
	done
done

# ARM
ARMS=(5 6 7)
for v in ${ARMS[@]}; do
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=$v go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o ./release/${name}_arm$v .
done
if $UPX; then upx -9 ./release/${name}_arm*; fi
tar -C ./release -zcf ./release/${name}_arm-$VERSION.tar.gz $(for v in ${ARMS[@]}; do echo -n "./${name}_arm$v ";done)
$MD5 ./release/${name}_arm-$VERSION.tar.gz

# MIPS # go 1.8+ required
LDFLAGS="-X main.version=$VERSION -s -w"
env CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o ./release/${name}_mipsle .
env CGO_ENABLED=0 GOOS=linux GOARCH=mips go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o ./release/${name}_mips .

if $UPX; then upx -9 ./release/${name}_mips**; fi
tar -C ./release -zcf ./release/${name}_mipsle-$VERSION.tar.gz ./${name}_mipsle
tar -C ./release -zcf ./release/${name}_mips-$VERSION.tar.gz ./${name}_mips
$MD5 ./release/${name}_mipsle-$VERSION.tar.gz
