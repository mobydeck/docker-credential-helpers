set shell := ["bash","-euxo","pipefail"]

release_dir := "release"
package_path := "github.com/docker/docker-credential-helpers"
helpers_darwin := "osxkeychain pass plain"
helpers_linux := "pass secretservice plain"
archs := "amd64 arm64"

build-release:
	#!/bin/bash
	rm -rf {{release_dir}}
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

publish-release: build-release
	#!/bin/bash
	if ! command -v gh >/dev/null; then
		echo "gh CLI must be installed to publish a release"
		exit 1
	fi
	gh_token="${GH_TOKEN:-}"
	if [ -z "$gh_token" ] && [ -f "$HOME/.netrc" ]; then
		gh_token=$(
			awk '
			{
				for (i = 1; i <= NF; i++) {
					tok = $i
					if (tok == "machine") {
						state = "machine"
						continue
					}
					if (tok == "login") {
						state = "login"
						continue
					}
					if (tok == "password") {
						state = "password"
						continue
					}
					if (state == "machine") {
						machine = tok
						state = ""
						continue
					}
					if (state == "password") {
						if (machine == "github.com") {
							print tok
							exit
						}
						state = ""
						continue
					}
				}
			}
			' "$HOME/.netrc"
		)
	fi
	if [ -n "$gh_token" ]; then
		export GH_TOKEN="$gh_token"
	fi
	latest_tag=$(git describe --tags --abbrev=0)
	if [ -z "$latest_tag" ]; then
		echo "no git tags available to release"
		exit 1
	fi
	assets=({{release_dir}}/*)
	if [ "${#assets[@]}" -eq 0 ]; then
		echo "no binaries found in {{release_dir}}; run just build-release first"
		exit 1
	fi
	gh repo set-default
	gh release create "$latest_tag" --title "$latest_tag" --notes "Release $latest_tag binaries" "${assets[@]}"
