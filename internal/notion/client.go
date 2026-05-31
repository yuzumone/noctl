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
// We use a raw request here because the notionapi library (v1.13.3+) has a bug where
// an empty SearchFilter is always serialized, causing validation errors in Notion,
// and it doesn't support the new "data_source" object type in 2026-03-11.
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
	req.Header.Set("Notion-Version", "2026-03-11")
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(b))
	}

	var rawSearchResp struct {
		Object     notionapi.ObjectType `json:"object"`
		Results    []json.RawMessage    `json:"results"`
		HasMore    bool                 `json:"has_more"`
		NextCursor notionapi.Cursor     `json:"next_cursor"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawSearchResp); err != nil {
		return nil, err
	}

	results := make([]notionapi.Object, 0, len(rawSearchResp.Results))
	for _, raw := range rawSearchResp.Results {
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		objectType := obj["object"].(string)
		switch objectType {
		case "page":
			var p notionapi.Page
			if err := json.Unmarshal(raw, &p); err == nil {
				results = append(results, &p)
			}
		case "database":
			var db notionapi.Database
			if err := json.Unmarshal(raw, &db); err == nil {
				results = append(results, &db)
			}
		case "data_source":
			var ds struct {
				ID    string               `json:"id"`
				Title []notionapi.RichText `json:"title"`
				URL   string               `json:"url"`
			}
			if err := json.Unmarshal(raw, &ds); err == nil {
				results = append(results, &notionapi.Database{
					Object: notionapi.ObjectTypeDatabase,
					ID:     notionapi.ObjectID(ds.ID),
					Title:  ds.Title,
					URL:    ds.URL,
				})
			}
		}
	}

	return &notionapi.SearchResponse{
		Object:     rawSearchResp.Object,
		Results:    results,
		HasMore:    rawSearchResp.HasMore,
		NextCursor: rawSearchResp.NextCursor,
	}, nil
}

// AppendChildren appends blocks to a parent block using the position object (required for 2026-03-11).
func (c *Client) AppendChildren(ctx context.Context, parentID string, children []notionapi.Block, afterID string) (*notionapi.AppendBlockChildrenResponse, error) {
	type position struct {
		Type       string `json:"type"`
		AfterBlock *struct {
			ID string `json:"id"`
		} `json:"after_block,omitempty"`
	}

	body := map[string]interface{}{
		"children": children,
	}

	if afterID != "" {
		body["position"] = position{
			Type: "after_block",
			AfterBlock: &struct {
				ID string `json:"id"`
			}{ID: afterID},
		}
	} else {
		body["position"] = position{
			Type: "end",
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children", parentID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", "2026-03-11")
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("append children failed with status %d: %s", resp.StatusCode, string(b))
	}

	var appendResp notionapi.AppendBlockChildrenResponse
	if err := json.NewDecoder(resp.Body).Decode(&appendResp); err != nil {
		return nil, err
	}

	return &appendResp, nil
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
		Client: notionapi.NewClient(
			notionapi.Token(token),
			notionapi.WithHTTPClient(httpClient),
			notionapi.WithVersion("2026-03-11"),
		),
		token: token,
	}
}

// formatID ensures a 32-character UUID has dashes in the correct places.
func formatID(id string) string {
	if len(id) != 32 {
		return id
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", id[0:8], id[8:12], id[12:16], id[16:20], id[20:])
}
