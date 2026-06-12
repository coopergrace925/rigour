package api

import (
	"regexp"
	"strconv"
	"strings"
)

type QueryFilters struct {
	FreeText []string
	Port     int
	ASN      int
	Country  string
	City     string
	Org      string
	CVE      string
	CPE      string
	Product  string
	Banner   string
	Title    string
	Server   string
}

var dorkRegex = regexp.MustCompile(`([a-zA-Z0-9_.-]+):(?:"([^"]+)"|([^\s]+))`)

func ParseShodanQuery(query string) QueryFilters {
	var filters QueryFilters

	matches := dorkRegex.FindAllStringSubmatch(query, -1)

	remainingQuery := query
	for _, match := range matches {
		fullMatch := match[0]
		key := strings.ToLower(match[1])
		val := match[2]
		if val == "" {
			val = match[3]
		}

		remainingQuery = strings.Replace(remainingQuery, fullMatch, "", 1)

		switch key {
		case "port":
			if p, err := strconv.Atoi(val); err == nil {
				filters.Port = p
			}
		case "asn":
			val = strings.TrimPrefix(strings.ToUpper(val), "AS")
			if a, err := strconv.Atoi(val); err == nil {
				filters.ASN = a
			}
		case "country":
			filters.Country = strings.ToUpper(val)
		case "city":
			filters.City = val
		case "org":
			filters.Org = val
		case "cve":
			filters.CVE = strings.ToUpper(val)
		case "cpe":
			filters.CPE = val
		case "product":
			filters.Product = val
		case "banner":
			filters.Banner = val
		case "title":
			filters.Title = val
		case "server":
			filters.Server = val
		}
	}

	words := strings.Fields(remainingQuery)
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			filters.FreeText = append(filters.FreeText, w)
		}
	}

	return filters
}
