set shell := ["bash","-euxo","pipefail"]

release_dir := "release"
package_path := "github.com/docker/docker-credential-helpers"
helpers_darwin := "osxkeychain pass plain"
helpers_linux := "pass secretservice plain"
archs := "amd64 arm64"

build-release:
	#!/bin/bash
	mkdir -p {{release_dir}}
	for os in darwin linux; do
		helpers="{{helpers_darwin}}"
		if [ "$os" = "linux" ]; then
			helpers="{{helpers_linux}}"
		fi
		for arch in {{archs}}; do
			for helper in $helpers; do
				bin="docker-credential-$helper"
				out={{release_dir}}/${bin}_${os}_${arch}
				GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-s -w -X {{package_path}}/credentials.Name=$bin" -o "$out" "./$helper/cmd/"
			done
		done
	done

default: build-release
