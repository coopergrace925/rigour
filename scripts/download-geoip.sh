#!/bin/bash
set -e

GEOIP_DIR="${GEOIP_DIR:-/data/geoip}"
MAXMIND_LICENSE_KEY="${MAXMIND_LICENSE_KEY}"

if [ -z "$MAXMIND_LICENSE_KEY" ]; then
    echo "Error: MAXMIND_LICENSE_KEY environment variable not set"
    echo "Get a free license key at: https://www.maxmind.com/en/geolite2/signup"
    exit 1
fi

mkdir -p "$GEOIP_DIR"
cd "$GEOIP_DIR"

echo "Downloading MaxMind GeoLite2 databases..."

# Download GeoLite2-City
wget -O GeoLite2-City.tar.gz \
  "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
tar -xzf GeoLite2-City.tar.gz --strip-components=1
rm GeoLite2-City.tar.gz

# Download GeoLite2-ASN
wget -O GeoLite2-ASN.tar.gz \
  "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
tar -xzf GeoLite2-ASN.tar.gz --strip-components=1
rm GeoLite2-ASN.tar.gz

echo "GeoIP databases downloaded to $GEOIP_DIR"
ls -lh "$GEOIP_DIR"/*.mmdb
