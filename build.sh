#!/bin/bash

app_name="openheart-protocol"
module_name="openheart.tylery.com/cmd/api"

if [ -n "$1" ];
then
  os_archs=("$1")
else
# Operating systems and architectures to build for
  os_archs=(
    "darwin/amd64"
    "darwin/arm64"
    "dragonfly/amd64"
    "freebsd/386"
    "freebsd/amd64"
    "freebsd/arm"
    "linux/386"
    "linux/amd64"
    "linux/arm"
    "linux/arm64"
    "linux/ppc64"
    "linux/ppc64le"
    "linux/mips"
    "linux/mipsle"
    "linux/mips64"
    "linux/mips64le"
    "netbsd/386"
    "netbsd/amd64"
    "netbsd/arm"
    "openbsd/386"
    "openbsd/amd64"
    "openbsd/arm"
    "plan9/386"
    "plan9/amd64"
    "solaris/amd64"
    "windows/386"
    "windows/amd64"
  )
fi


# Loop through each OS/architecture combination
mkdir -p build
echo "Building..."
for os_arch in "${os_archs[@]}"; do
  # Split the string into OS and architecture
  os=$(echo "$os_arch" | cut -d/ -f1)
  arch=$(echo "$os_arch" | cut -d/ -f2)

  # Set GOOS and GOARCH environment variables
  export GOOS="$os"
  export GOARCH="$arch"

  # Set the output file name (e.g., myapp-darwin-amd64)
  output_file="$app_name-$os-$arch"
  if [[ "$os" == "windows" ]]; then
    output_file="$output_file.exe" # Add .exe extension for Windows
  fi

  # Build!
  echo -en "\t$os/$arch: $output_file"
  go build -ldflags "-s -w" -o build/"$output_file" "$module_name"
  gzip -f build/"$output_file"

  echo " - Complete!"
done

echo "Build complete!"

