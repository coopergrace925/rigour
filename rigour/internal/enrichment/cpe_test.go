package enrichment

import (
	"testing"
)

func TestCPEMatchRules(t *testing.T) {
	matcher := NewCPEMatcher()

	cpe, product, version := matcher.Match("http", "HTTP/1.1 200 OK\r\nServer: Apache/2.4.41")
	if cpe != "cpe:/a:apache:http_server:2.4.41" {
		t.Errorf("Expected Apache CPE, got %s", cpe)
	}
	if product != "Apache HTTPD" {
		t.Errorf("Expected Apache HTTPD, got %s", product)
	}
	if version != "2.4.41" {
		t.Errorf("Expected version 2.4.41, got %s", version)
	}

	cpe, product, version = matcher.Match("ssh", "SSH-2.0-OpenSSH_8.9p1")
	if cpe != "cpe:/a:openbsd:openssh:8.9" {
		t.Errorf("Expected OpenSSH CPE, got %s", cpe)
	}
	if expected := "OpenSSH"; product != expected {
		t.Errorf("Expected %s, got %s", expected, product)
	}
	if expected := "8.9"; version != expected {
		t.Errorf("Expected version %s, got %s", expected, version)
	}
}
