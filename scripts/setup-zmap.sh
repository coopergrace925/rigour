#!/bin/bash
set -e

echo "Installing ZMap..."

# Install dependencies
apt-get update
apt-get install -y build-essential cmake libgmp3-dev gengetopt libpcap-dev flex byacc libjson-c-dev pkg-config libunistring-dev

# Clone and build ZMap
cd /tmp
git clone https://github.com/zmap/zmap.git
cd zmap
cmake .
make -j$(nproc)
make install

# Verify installation
zmap --version

echo "ZMap installed successfully"
