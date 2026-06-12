package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ctrlsam/rigour/internal/opensearch"
	"github.com/ctrlsam/rigour/internal/storage"
	"github.com/ctrlsam/rigour/pkg/types"
)

type HostRepository struct {
	client *opensearch.Client
	index  string
}

func NewHostRepository(client *opensearch.Client) *HostRepository {
	return &HostRepository{
		client: client,
		index:  "hosts",
	}
}

func (r *HostRepository) EnsureHost(ctx context.Context, ip string, now time.Time) error {
	return fmt.Errorf("EnsureHost not implemented for OpenSearch")
}

func (r *HostRepository) UpsertService(ctx context.Context, svc types.Service) error {
	return fmt.Errorf("UpsertService not implemented for OpenSearch")
}

func (r *HostRepository) UpdateHost(ctx context.Context, host types.Host) error {
	return fmt.Errorf("UpdateHost not implemented for OpenSearch")
}

func (r *HostRepository) GetByIP(ctx context.Context, ip string) (*types.Host, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"ip": ip,
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	res, err := r.client.Raw().Search(
		r.client.Raw().Search.WithContext(ctx),
		r.client.Raw().Search.WithIndex(r.index),
		r.client.Raw().Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("OpenSearch error: %s", res.String())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source types.HostDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Hits.Hits) == 0 {
		return nil, fmt.Errorf("host not found")
	}

	return convertHostDocumentToHost(result.Hits.Hits[0].Source), nil
}

func (r *HostRepository) Search(ctx context.Context, filter map[string]interface{}, lastID string, limit int) ([]types.Host, string, error) {
	query := buildOpenSearchQuery(filter)

	searchBody := map[string]interface{}{
		"query": query,
		"size":  limit,
		"sort": []map[string]interface{}{
			{"last_seen": map[string]string{"order": "desc"}},
			{"_id": map[string]string{"order": "asc"}},
		},
	}

	if lastID != "" {
		searchBody["search_after"] = []interface{}{lastID}
	}

	body, err := json.Marshal(searchBody)
	if err != nil {
		return nil, "", err
	}

	res, err := r.client.Raw().Search(
		r.client.Raw().Search.WithContext(ctx),
		r.client.Raw().Search.WithIndex(r.index),
		r.client.Raw().Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, "", fmt.Errorf("OpenSearch error: %s", string(bodyBytes))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				ID     string              `json:"_id"`
				Source types.HostDocument `json:"_source"`
				Sort   []interface{}       `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	hosts := make([]types.Host, 0, len(result.Hits.Hits))
	var nextID string

	for _, hit := range result.Hits.Hits {
		hosts = append(hosts, *convertHostDocumentToHost(hit.Source))
		nextID = hit.ID
	}

	return hosts, nextID, nil
}

func (r *HostRepository) Facets(ctx context.Context, filter map[string]interface{}) (*storage.FacetCounts, error) {
	query := buildOpenSearchQuery(filter)

	aggs := map[string]interface{}{
		"services": map[string]interface{}{
			"nested": map[string]interface{}{
				"path": "ports",
			},
			"aggs": map[string]interface{}{
				"service_facet": map[string]interface{}{
					"terms": map[string]interface{}{
						"field": "ports.protocol",
						"size":  50,
					},
				},
			},
		},
		"countries": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "country",
				"size":  50,
			},
		},
		"asns": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "asn",
				"size":  50,
			},
		},
	}

	searchBody := map[string]interface{}{
		"query": query,
		"size":  0,
		"aggs":  aggs,
	}

	body, err := json.Marshal(searchBody)
	if err != nil {
		return nil, err
	}

	res, err := r.client.Raw().Search(
		r.client.Raw().Search.WithContext(ctx),
		r.client.Raw().Search.WithIndex(r.index),
		r.client.Raw().Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("OpenSearch error: %s", res.String())
	}

	var result struct {
		Aggregations map[string]interface{} `json:"aggregations"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	facets := &storage.FacetCounts{
		Services:  make(map[string]int),
		Countries: []storage.CountryFacet{},
		ASNs:      []storage.ASNFacet{},
	}

	if services, ok := result.Aggregations["services"].(map[string]interface{}); ok {
		if serviceFacet, ok := services["service_facet"].(map[string]interface{}); ok {
			if buckets, ok := serviceFacet["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					b := bucket.(map[string]interface{})
					key := b["key"].(string)
					count := int(b["doc_count"].(float64))
					facets.Services[key] = count
				}
			}
		}
	}

	if countries, ok := result.Aggregations["countries"].(map[string]interface{}); ok {
		if buckets, ok := countries["buckets"].([]interface{}); ok {
			for _, bucket := range buckets {
				b := bucket.(map[string]interface{})
				code := b["key"].(string)
				count := int(b["doc_count"].(float64))
				facets.Countries = append(facets.Countries, storage.CountryFacet{
					Code:  code,
					Name:  code,
					Count: count,
				})
			}
		}
	}

	if asns, ok := result.Aggregations["asns"].(map[string]interface{}); ok {
		if buckets, ok := asns["buckets"].([]interface{}); ok {
			for _, bucket := range buckets {
				b := bucket.(map[string]interface{})
				code := uint32(b["key"].(float64))
				count := int(b["doc_count"].(float64))
				facets.ASNs = append(facets.ASNs, storage.ASNFacet{
					Code:  code,
					Name:  fmt.Sprintf("AS%d", code),
					Count: count,
				})
			}
		}
	}

	return facets, nil
}

func buildOpenSearchQuery(filter map[string]interface{}) map[string]interface{} {
	if len(filter) == 0 {
		return map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	must := []map[string]interface{}{}

	for key, value := range filter {
		parts := strings.Split(key, ".")
		if parts[0] == "services" {
			nested := map[string]interface{}{
				"nested": map[string]interface{}{
					"path": "ports",
					"query": map[string]interface{}{
						"term": map[string]interface{}{
							fmt.Sprintf("ports.%s", strings.Join(parts[1:], ".")): value,
						},
					},
				},
			}
			must = append(must, nested)
		} else {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{
					key: value,
				},
			})
		}
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"must": must,
		},
	}
}

func convertHostDocumentToHost(doc types.HostDocument) *types.Host {
	services := make([]types.Service, 0, len(doc.Ports))
	for _, port := range doc.Ports {
		svc := types.Service{
			IP:        doc.IP,
			Port:      port.Port,
			Protocol:  port.Protocol,
			Transport: "tcp",
			LastScan:  port.LastSeen,
		}
		services = append(services, svc)
	}

	return &types.Host{
		ID:    doc.IP,
		IP:    doc.IP,
		IPInt: doc.IPInt,
		ASN: &types.ASNInfo{
			Number:       uint32(doc.ASN),
			Organization: doc.Org,
		},
		Location: &types.Location{
			City:        doc.City,
			CountryCode: doc.Country,
			CountryName: doc.Country,
		},
		FirstSeen: doc.LastSeen,
		LastSeen:  doc.LastSeen,
		Services:  services,
	}
}
