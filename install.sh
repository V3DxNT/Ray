#!/bin/bash

os=$(uname -s)
arch=$(uname -m)

if [ "$os" == "Darwin" ]; then
  GOOS_VAR="darwin"
elif [ "$os" == "Linux" ]; then
  GOOS_VAR="linux"
else
  echo "Unsupported OS"
  exit 1
fi

if [ "$arch" == "x86_64" ]; then
  GOARCH_VAR="amd64"
elif [ "$arch" == "arm64" ] || [ "$arch" == "aarch64" ]; then
  GOARCH_VAR="arm64"
else
  echo "Unsupported Architecture"
  exit 1
fi

if [ -f "cmd/main.go" ] && [ -f "go.mod" ]; then
  echo "Local repository detected. Building directly..."

  GOOS=$GOOS_VAR GOARCH=$GOARCH_VAR go build -o ray cmd/main.go

  sudo mv ray /usr/local/bin/ray
  sudo chmod +x /usr/local/bin/ray

else
  echo "No local repository found. Downloading ray..."

  TMP_DIR=$(mktemp -d)
  git clone --depth 1 https://github.com/V3DxNT/ray.git "$TMP_DIR"

  cd "$TMP_DIR" || exit

  GOOS=$GOOS_VAR GOARCH=$GOARCH_VAR go build -o ray cmd/main.go

  sudo mv ray /usr/local/bin/ray
  sudo chmod +x /usr/local/bin/ray

  cd - > /dev/null || exit
  rm -rf "$TMP_DIR"
fi

echo -e "\033[36m"
echo "██████╗  █████╗ ██╗   ██╗"
echo "██╔══██╗██╔══██╗╚██╗ ██╔╝"
echo "██████╔╝███████║ ╚████╔╝ "
echo "██╔══██╗██╔══██║  ╚██╔╝  "
echo "██║  ██║██║  ██║   ██║   "
echo "╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   "
echo -e "\033[0m"

echo "✨ Installation Successful!"
echo "Run 'ray' in any directory to map the architecture."
echo "Built by: www.vedx.dev"