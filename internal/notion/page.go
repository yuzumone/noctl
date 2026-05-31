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

// QueryDatabaseAll fetches all pages from a specific database by iterating through all pages.
func (c *Client) QueryDatabaseAll(ctx context.Context, dbID string) ([]notionapi.Page, error) {
	var pages []notionapi.Page
	var cursor notionapi.Cursor

	for {
		res, err := c.QueryDatabase(ctx, dbID, cursor)
		if err != nil {
			return nil, err
		}
		pages = append(pages, res.Results...)
		if !res.HasMore {
			break
		}
		cursor = res.NextCursor
	}

	return pages, nil
}

// QueryDatabase fetches pages from a specific database (or data source).
func (c *Client) QueryDatabase(ctx context.Context, id string, cursor notionapi.Cursor) (*notionapi.DatabaseQueryResponse, error) {
	id = formatID(id)
	body := map[string]interface{}{
		"page_size": 100,
	}
	if cursor != "" {
		body["start_cursor"] = cursor
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	// Helper to perform the actual query
	doQuery := func(endpoint string, targetID string) (*notionapi.DatabaseQueryResponse, int, []byte, error) {
		url := fmt.Sprintf("https://api.notion.com/v1/%s/%s/query", endpoint, targetID)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, 0, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Notion-Version", "2026-03-11")
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, 0, nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, resp.StatusCode, b, nil
		}

		var res notionapi.DatabaseQueryResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, resp.StatusCode, nil, err
		}
		return &res, resp.StatusCode, nil, nil
	}

	// 1. Try querying as a data_source first
	res, status, _, err := doQuery("data_sources", id)
	if err == nil && status == http.StatusOK {
		return res, nil
	}

	// 2. If it failed with 400 or 404, it might be a database_id.
	// We need to fetch the database to get its data_source IDs.
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s", id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", "2026-03-11")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		var dbMetadata struct {
			DataSources []struct {
				ID string `json:"id"`
			} `json:"data_sources"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&dbMetadata); err == nil && len(dbMetadata.DataSources) > 0 {
			// Query the first data source for now.
			// Ideally we should query all and combine if we want a complete view,
			// but for single-source databases (the common case), the first one is enough.
			dsID := dbMetadata.DataSources[0].ID
			res, _, _, err := doQuery("data_sources", dsID)
			return res, err
		}
	}

	// 3. Fallback to the old databases endpoint as a last resort
	res, status, b, err := doQuery("databases", id)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("querying database/data_source %s failed: %s", id, string(b))
	}

	return res, nil
}

// CreatePage creates a new page in the specified database (or data source).
func (c *Client) CreatePage(ctx context.Context, dbID string, properties notionapi.Properties) (*notionapi.Page, error) {
	// Try data_source_id parent first
	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"type":           "data_source_id",
			"data_source_id": dbID,
		},
		"properties": properties,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.notion.com/v1/pages", bytes.NewBuffer(jsonBody))
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

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		// Fallback to database_id parent
		body["parent"] = map[string]interface{}{
			"type":        "database_id",
			"database_id": dbID,
		}
		jsonBody, _ = json.Marshal(body)
		req, _ = http.NewRequestWithContext(ctx, "POST", "https://api.notion.com/v1/pages", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Notion-Version", "2026-03-11")
		req.Header.Set("Content-Type", "application/json")
		resp, err = httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("creating page failed: status %d: %s", resp.StatusCode, string(b))
		}
	}
	defer func() { _ = resp.Body.Close() }()

	var page notionapi.Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	return &page, nil
}

// UpdatePage updates an existing page.
func (c *Client) UpdatePage(ctx context.Context, pageID string, properties notionapi.Properties) (*notionapi.Page, error) {
	req := &notionapi.PageUpdateRequest{
		Properties: properties,
	}

	res, err := c.Page.Update(ctx, notionapi.PageID(pageID), req)
	if err != nil {
		return nil, fmt.Errorf("updating page %s: %w", pageID, err)
	}

	return res, nil
}
