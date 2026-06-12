package opensearch

import (
	"crypto/tls"
	"fmt"
	"net/http"

	opensearchgo "github.com/opensearch-project/opensearch-go/v2"
)

type Client struct {
	os      *opensearchgo.Client
	baseURL string
}

func NewClient(addresses []string) (*Client, error) {
	cfg := opensearchgo.Config{
		Addresses: addresses,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// Dev/test only - production should verify certificates
				InsecureSkipVerify: true,
			},
		},
	}

	client, err := opensearchgo.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	// Verify connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("OpenSearch returned error: %s", res.Status())
	}

	baseURL := ""
	if len(addresses) > 0 {
		baseURL = addresses[0]
	}

	return &Client{os: client, baseURL: baseURL}, nil
}

func (c *Client) Raw() *opensearchgo.Client {
	return c.os
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Perform(req *http.Request) (*http.Response, error) {
	return c.os.Perform(req)
}
