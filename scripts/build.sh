#!/bin/bash

CGO_ENABLED=0
DIST_DIR="dist"
VERSION=${VERSION:-dev}

platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64" "windows/arm64")

for platform in "${platforms[@]}"
do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output="$DIST_DIR/$(basename $PWD)_${GOOS}_${GOARCH}"

  if [ "$GOOS" == "windows" ]; then
    output="$output.exe"
  fi

  echo "Building $output with version $VERSION"
  CGO_ENABLED=$CGO_ENABLED GOOS=$GOOS GOARCH=$GOARCH \
    go build -gcflags="all=-l -B" -trimpath -ldflags="-s -w -X main.version=$VERSION" -o "$output" ./cmd/tgeraser/
done
