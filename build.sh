#!/bin/bash

PLATFORMS=(
    "linux/amd64"    # Linux 64-bit
    "linux/386"      # Linux 32-bit
    "linux/arm64"    # Linux ARM 64-bit
    "linux/arm"      # Linux ARM 32-bit
    "darwin/amd64"   # macOS Intel 64-bit
    "darwin/arm64"   # macOS Apple Silicon (M1/M2)
    "freebsd/amd64"  # FreeBSD 64-bit
    "freebsd/386"    # FreeBSD 32-bit
    "openbsd/amd64"  # OpenBSD 64-bit
    "openbsd/386"    # OpenBSD 32-bit
)
OUTPUT_DIR="./bin"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")

mkdir -p $OUTPUT_DIR

for PLATFORM in "${PLATFORMS[@]}"
do
  OS=$(echo $PLATFORM | cut -d'/' -f1)
  ARCH=$(echo $PLATFORM | cut -d'/' -f2)
  OUTPUT_NAME=$OUTPUT_DIR/schedulr-$OS-$ARCH-$TIMESTAMP

  if [ $OS = "windows" ]; then
    OUTPUT_NAME+='.exe'
  fi

  echo "Building for $OS/$ARCH..."
  GOOS=$OS GOARCH=$ARCH go build -o $OUTPUT_NAME .

  if [ $? -ne 0 ]; then
    echo "Error: Failed to build for $OS/$ARCH"
    exit 1
  fi
done

echo "Build completed successfully."