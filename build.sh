#!/usr/bin/env sh
set -eu

appname="trail"

version="$(
	sed -n 's/^const version = "\([^"]*\)"/\1/p' main.go | head -n 1
)"
if [ -z "$version" ]; then
	echo "Warning: Could not detect version from main.go, using default" >&2
	version="0.0.0"
else
	echo "Detected version: $version"
fi

if ! command -v zip >/dev/null 2>&1; then
	echo "zip is required to create release archives" >&2
	exit 1
fi

mkdir -p release

for target in \
	windows_amd64 \
	windows_arm64 \
	linux_amd64 \
	linux_arm64 \
	darwin_amd64 \
	darwin_arm64
do
	goos="${target%_*}"
	goarch="${target#*_}"
	build_dir="build/${appname}_${target}"
	output_file="${build_dir}/${appname}"

	if [ "$goos" = "windows" ]; then
		output_file="${output_file}.exe"
	fi

	echo "Building for $target..."
	rm -rf "$build_dir"
	mkdir -p "$build_dir"

	GOOS="$goos" GOARCH="$goarch" go build -o "$output_file" -ldflags "-s -w" main.go

	if [ "$goos" = "windows" ]; then
		cp installer.ps1 "$build_dir/"
	fi

	zip_path="release/${appname}_${version}_${target}.zip"
	rm -f "$zip_path"

	echo "Creating ZIP: $zip_path"
	(
		cd "$build_dir"
		zip -qr "../../$zip_path" .
	)

	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$zip_path"
	elif command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$zip_path"
	else
		echo "SHA256 skipped: shasum or sha256sum not found" >&2
	fi
done

echo "Build completed! Files are in the release directory."
