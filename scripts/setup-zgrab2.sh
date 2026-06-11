#!/bin/bash
set -e

echo "Installing ZGrab2..."

# Check if already installed
if command -v zgrab2 &> /dev/null; then
    echo "ZGrab2 already installed: $(zgrab2 --version)"
    exit 0
fi

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "Go not found, installing..."
    wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
fi

# Clone and build ZGrab2
cd /tmp
git clone https://github.com/zmap/zgrab2.git
cd zgrab2
make
cp zgrab2 /usr/local/bin/

# Cleanup
rm -rf /tmp/zgrab2

# Verify installation
zgrab2 --version

echo "ZGrab2 installed successfully"
