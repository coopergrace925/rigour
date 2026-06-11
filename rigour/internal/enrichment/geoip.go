package enrichment

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type GeoResult struct {
	Country string `json:"country"`
	City    string `json:"city"`
	ASN     int    `json:"asn"`
	Org     string `json:"org"`
	Lat     float64 `json:"lat,omitempty"`
	Lon     float64 `json:"lon,omitempty"`
}

type GeoIPLookup struct {
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
}

func NewGeoIPLookup(cityDBPath, asnDBPath string) (*GeoIPLookup, error) {
	cityDB, err := geoip2.Open(cityDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoLite2-City DB: %w", err)
	}

	asnDB, err := geoip2.Open(asnDBPath)
	if err != nil {
		cityDB.Close()
		return nil, fmt.Errorf("failed to open GeoLite2-ASN DB: %w", err)
	}

	return &GeoIPLookup{
		cityDB: cityDB,
		asnDB:  asnDB,
	}, nil
}

func (g *GeoIPLookup) Lookup(ipStr string) GeoResult {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoResult{}
	}

	result := GeoResult{}

	// City/Country lookup
	city, err := g.cityDB.City(ip)
	if err == nil {
		result.Country = city.Country.IsoCode
		if len(city.City.Names) > 0 {
			result.City = city.City.Names["en"]
		}
		result.Lat = city.Location.Latitude
		result.Lon = city.Location.Longitude
	}

	// ASN lookup
	asn, err := g.asnDB.ASN(ip)
	if err == nil {
		result.ASN = int(asn.AutonomousSystemNumber)
		result.Org = asn.AutonomousSystemOrganization
	}

	return result
}

func (g *GeoIPLookup) Close() {
	if g.cityDB != nil {
		g.cityDB.Close()
	}
	if g.asnDB != nil {
		g.asnDB.Close()
	}
}
