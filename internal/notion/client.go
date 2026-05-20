package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jomei/notionapi"
)

// Client is a wrapper around notionapi.Client.
type Client struct {
	*notionapi.Client
	token string
}

// retryTransport is a http.RoundTripper that retries on 429 and 5xx errors.
type retryTransport struct {
	base http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < 3; i++ {
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate limited, wait and retry
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		if resp.StatusCode >= 500 {
			// Server error, wait and retry
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}

		return resp, nil
	}

	return resp, err
}

// GlobalSearch searches for pages and databases by title.
// We use a raw request here because the notionapi library (v1.13.3) has a bug where
// an empty SearchFilter is always serialized, causing validation errors in Notion.
func (c *Client) GlobalSearch(ctx context.Context, query string, cursor notionapi.Cursor) (*notionapi.SearchResponse, error) {
	body := map[string]interface{}{
		"query":     query,
		"page_size": 100,
	}
	if cursor != "" {
		body["start_cursor"] = cursor
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.notion.com/v1/search", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", "2022-06-28")
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(b))
	}

	var searchResp notionapi.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return &searchResp, nil
}

// NewClient creates a new Notion client with retry logic.
func NewClient(token string) *Client {
	httpClient := &http.Client{
		Transport: &retryTransport{
			base: http.DefaultTransport,
		},
		Timeout: 30 * time.Second,
	}

	return &Client{
		Client: notionapi.NewClient(notionapi.Token(token), notionapi.WithHTTPClient(httpClient)),
		token:  token,
	}
}
