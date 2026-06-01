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
	token      string
	httpClient *http.Client
}

// retryTransport is a http.RoundTripper that retries on 429 and 5xx errors.
type retryTransport struct {
	base        http.RoundTripper
	maxAttempts int
	sleep       func(time.Duration)
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	attempts := t.maxAttempts
	if attempts == 0 {
		attempts = 3
	}

	for i := 0; i < attempts; i++ {
		if i > 0 && req.Body != nil && req.Body != http.NoBody {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if !isRetryableStatus(resp.StatusCode) || i == attempts-1 || !canRetryRequest(req) {
			return resp, nil
		}

		closeResponseBody(resp)
		t.wait(retryDelay(resp.StatusCode, i))
	}

	return resp, err
}

func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func canRetryRequest(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

func closeResponseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func retryDelay(status int, attempt int) time.Duration {
	if status == http.StatusTooManyRequests {
		return time.Duration(attempt+1) * time.Second
	}
	return time.Duration(attempt+1) * 500 * time.Millisecond
}

func (t *retryTransport) wait(delay time.Duration) {
	if t.sleep != nil {
		t.sleep(delay)
		return
	}
	time.Sleep(delay)
}

func (c *Client) doNotionJSON(ctx context.Context, method string, path string, requestBody interface{}, responseBody interface{}, operation string) error {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, "https://api.notion.com/v1"+path, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", "2026-03-11")
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s failed with status %d: %s", operation, resp.StatusCode, string(b))
	}

	if responseBody == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(responseBody)
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

	var rawSearchResp struct {
		Object     notionapi.ObjectType `json:"object"`
		Results    []json.RawMessage    `json:"results"`
		HasMore    bool                 `json:"has_more"`
		NextCursor notionapi.Cursor     `json:"next_cursor"`
	}

	if err := c.doNotionJSON(ctx, http.MethodPost, "/search", body, &rawSearchResp, "search"); err != nil {
		return nil, err
	}

	results := make([]notionapi.Object, 0, len(rawSearchResp.Results))
	for _, raw := range rawSearchResp.Results {
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		objectType, ok := obj["object"].(string)
		if !ok {
			continue
		}

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

	var appendResp notionapi.AppendBlockChildrenResponse
	path := fmt.Sprintf("/blocks/%s/children", parentID)
	if err := c.doNotionJSON(ctx, http.MethodPatch, path, body, &appendResp, "append children"); err != nil {
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
		token:      token,
		httpClient: httpClient,
	}
}

// formatID ensures a 32-character UUID has dashes in the correct places.
func formatID(id string) string {
	if len(id) != 32 {
		return id
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", id[0:8], id[8:12], id[12:16], id[16:20], id[20:])
}
