package opensearch

import (
	"fmt"
	"strings"
)

const HostsIndex = "hosts"

const hostsMapping = `{
  "settings": {
    "number_of_shards": 6,
    "number_of_replicas": 1,
    "refresh_interval": "30s",
    "index.routing.allocation.total_shards_per_node": 4
  },
  "mappings": {
    "dynamic": "strict",
    "properties": {
      "ip":        {"type": "ip"},
      "ip_int":    {"type": "long"},
      "asn":       {"type": "integer"},
      "org":       {"type": "keyword"},
      "country":   {"type": "keyword"},
      "city":      {"type": "keyword"},
      "rdns":      {"type": "keyword"},
      "last_seen": {"type": "date"},
      "is_stale":  {"type": "boolean"},
      "cves":      {"type": "keyword"},
      "tags":      {"type": "keyword"},
      "ports": {
        "type": "nested",
        "properties": {
          "port":      {"type": "integer"},
          "protocol":  {"type": "keyword"},
          "service":   {"type": "keyword"},
          "product":   {"type": "keyword"},
          "cpe":       {"type": "keyword"},
          "banner":    {"type": "text", "analyzer": "standard"},
          "last_seen": {"type": "date"},
          "http": {
            "properties": {
              "status_code": {"type": "integer"},
              "title":       {"type": "text"},
              "server":      {"type": "keyword"}
            }
          },
          "tls": {
            "properties": {
              "version": {"type": "keyword"},
              "cert": {
                "properties": {
                  "subject_cn":  {"type": "keyword"},
                  "issuer_cn":   {"type": "keyword"},
                  "fingerprint": {"type": "keyword"},
                  "not_after":   {"type": "date"},
                  "san":         {"type": "keyword"}
                }
              }
            }
          },
          "ssh": {
            "properties": {
              "hassh":     {"type": "keyword"},
              "server_id": {"type": "keyword"},
              "kex_algos": {"type": "keyword"}
            }
          }
        }
      }
    }
  }
}`

func (c *Client) CreateHostsIndex() error {
	res, err := c.os.Indices.Exists([]string{HostsIndex})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	// Index already exists
	if !res.IsError() {
		return nil
	}

	// Create the index
	res, err = c.os.Indices.Create(
		HostsIndex,
		c.os.Indices.Create.WithBody(strings.NewReader(hostsMapping)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	return nil
}
