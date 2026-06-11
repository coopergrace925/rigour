package enrichment

import (
	"regexp"
	"strings"
)

type MatchRule struct {
	Service     string
	Regex       *regexp.Regexp
	ProductName string
	VersionTpl  string
	CPETpl      string
}

type CPEMatcher struct {
	Rules []MatchRule
}

func NewCPEMatcher() *CPEMatcher {
	rules := []MatchRule{
		{
			Service:     "http",
			Regex:       regexp.MustCompile(`Server: Apache/([0-9.]+)`),
			ProductName: "Apache HTTPD",
			VersionTpl:  "$1",
			CPETpl:      "cpe:/a:apache:http_server:$1",
		},
		{
			Service:     "http",
			Regex:       regexp.MustCompile(`Server: nginx/([0-9.]+)`),
			ProductName: "nginx",
			VersionTpl:  "$1",
			CPETpl:      "cpe:/a:f5:nginx:$1",
		},
		{
			Service:     "ssh",
			Regex:       regexp.MustCompile(`SSH-2.0-OpenSSH_([0-9.]+)`),
			ProductName: "OpenSSH",
			VersionTpl:  "$1",
			CPETpl:      "cpe:/a:openbsd:openssh:$1",
		},
	}
	return &CPEMatcher{Rules: rules}
}

func (m *CPEMatcher) Match(service string, banner string) (cpe string, product string, version string) {
	for _, rule := range m.Rules {
		if strings.ToLower(rule.Service) != strings.ToLower(service) {
			continue
		}

		matches := rule.Regex.FindStringSubmatch(banner)
		if len(matches) > 1 {
			version = strings.ReplaceAll(rule.VersionTpl, "$1", matches[1])
			cpe = strings.ReplaceAll(rule.CPETpl, "$1", matches[1])
			product = rule.ProductName
			return cpe, product, version
		}
	}
	return "", "", ""
}
