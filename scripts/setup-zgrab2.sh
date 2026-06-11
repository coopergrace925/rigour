#!/bin/bash
set -e

echo "Installing ZGrab2..."

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "Go not found, installing..."
    wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
fi

# Clone and build ZGrab2
cd /tmp
git clone https://github.com/zmap/zgrab2.git
cd zgrab2
make
cp zgrab2 /usr/local/bin/

# Verify installation
zgrab2 --version

echo "ZGrab2 installed successfully"
