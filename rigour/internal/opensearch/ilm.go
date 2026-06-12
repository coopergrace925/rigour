package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CreateILMPolicy creates an Index State Management policy for hot/warm/delete lifecycle
func (c *Client) CreateILMPolicy(ctx context.Context, policyName string) error {
	policy := map[string]interface{}{
		"policy": map[string]interface{}{
			"description": "Rigour hosts index lifecycle: hot -> warm -> delete",
			"default_state": "hot",
			"states": []map[string]interface{}{
				{
					"name": "hot",
					"actions": []map[string]interface{}{
						{
							"rollover": map[string]interface{}{
								"min_index_age": "1d",
								"min_primary_shard_size": "30gb",
							},
						},
					},
					"transitions": []map[string]interface{}{
						{
							"state_name": "warm",
							"conditions": map[string]interface{}{
								"min_index_age": "7d",
							},
						},
					},
				},
				{
					"name": "warm",
					"actions": []map[string]interface{}{
						{
							"replica_count": map[string]interface{}{
								"number_of_replicas": 1,
							},
						},
						{
							"force_merge": map[string]interface{}{
								"max_num_segments": 1,
							},
						},
					},
					"transitions": []map[string]interface{}{
						{
							"state_name": "delete",
							"conditions": map[string]interface{}{
								"min_index_age": "90d",
							},
						},
					},
				},
				{
					"name": "delete",
					"actions": []map[string]interface{}{
						{
							"delete": map[string]interface{}{},
						},
					},
					"transitions": []map[string]interface{}{},
				},
			},
		},
	}

	body, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal ILM policy: %w", err)
	}

	url := fmt.Sprintf("%s/_plugins/_ism/policies/%s", c.baseURL, policyName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.os.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to create ILM policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ILM policy creation failed: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

// AttachILMPolicy attaches an ISM policy to an index
func (c *Client) AttachILMPolicy(ctx context.Context, indexPattern, policyID string) error {
	payload := map[string]interface{}{
		"policy_id": policyID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal attach request: %w", err)
	}

	url := fmt.Sprintf("%s/_plugins/_ism/add/%s", c.baseURL, indexPattern)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.os.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to attach ILM policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ILM policy attach failed: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

// CreateRollingIndex creates a new time-based index with ILM policy
func (c *Client) CreateRollingIndex(ctx context.Context, baseName string) error {
	timestamp := time.Now().Format("2006-01-02")
	indexName := fmt.Sprintf("%s-%s-000001", baseName, timestamp)

	// Create index with alias
	indexBody := map[string]interface{}{
		"aliases": map[string]interface{}{
			baseName: map[string]interface{}{
				"is_write_index": true,
			},
		},
		"settings": map[string]interface{}{
			"index": map[string]interface{}{
				"number_of_shards":   6,
				"number_of_replicas": 2,
				"refresh_interval":   "30s",
			},
		},
	}

	body, err := json.Marshal(indexBody)
	if err != nil {
		return fmt.Errorf("failed to marshal index body: %w", err)
	}

	res, err := c.os.Indices.Create(
		indexName,
		c.os.Indices.Create.WithContext(ctx),
		c.os.Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return fmt.Errorf("failed to create rolling index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("rolling index creation failed: %s - %s", res.Status(), string(body))
	}

	return nil
}

// GetIndexStats returns statistics for indices
func (c *Client) GetIndexStats(ctx context.Context, indexPattern string) (map[string]interface{}, error) {
	res, err := c.os.Indices.Stats(
		c.os.Indices.Stats.WithContext(ctx),
		c.os.Indices.Stats.WithIndex(indexPattern),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("index stats request failed: %s", res.Status())
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return stats, nil
}

// ForceRollover manually triggers an index rollover
func (c *Client) ForceRollover(ctx context.Context, alias string) error {
	res, err := c.os.Indices.Rollover(
		alias,
		c.os.Indices.Rollover.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to rollover index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("rollover failed: %s - %s", res.Status(), string(body))
	}

	return nil
}

// GetILMPolicyStatus returns the current ISM policy status for an index
func (c *Client) GetILMPolicyStatus(ctx context.Context, indexName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/_plugins/_ism/explain/%s", c.baseURL, indexName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.os.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ISM status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ISM status request failed: %s", resp.Status)
	}

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status: %w", err)
	}

	return status, nil
}
