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

// GeoIPLookup provides thread-safe GeoIP and ASN lookups using MaxMind databases.
// Multiple goroutines may call Lookup() concurrently.
type GeoIPLookup struct {
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
}

// NewGeoIPLookup opens the GeoLite2-City and GeoLite2-ASN databases.
// Caller must call Close() when done to release database handles.
// Example:
//
//	lookup, err := NewGeoIPLookup(cityPath, asnPath)
//	if err != nil { return err }
//	defer lookup.Close()
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

// Lookup performs GeoIP and ASN lookups for the given IP address.
// Returns error if the IP address is invalid. Database lookup errors are
// not returned; the function returns partial results if available.
func (g *GeoIPLookup) Lookup(ipStr string) (GeoResult, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoResult{}, fmt.Errorf("invalid IP address: %s", ipStr)
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
	// Note: Not returning error here - we allow partial results

	// ASN lookup
	asn, err := g.asnDB.ASN(ip)
	if err == nil {
		result.ASN = int(asn.AutonomousSystemNumber)
		result.Org = asn.AutonomousSystemOrganization
	}

	return result, nil
}

// Close releases database handles. Must be called when done using GeoIPLookup.
func (g *GeoIPLookup) Close() {
	if g.cityDB != nil {
		g.cityDB.Close()
	}
	if g.asnDB != nil {
		g.asnDB.Close()
	}
}
