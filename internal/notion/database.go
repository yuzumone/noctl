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

// ListDatabases fetches all databases (now called data sources) accessible to the integration.
func (c *Client) ListDatabases(ctx context.Context) ([]notionapi.Database, error) {
	var databases []notionapi.Database
	var cursor notionapi.Cursor

	for {
		body := map[string]interface{}{
			"filter": map[string]interface{}{
				"value":    "data_source",
				"property": "object",
			},
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
			return nil, fmt.Errorf("search databases failed with status %d: %s", resp.StatusCode, string(b))
		}

		var searchResp struct {
			Results []json.RawMessage `json:"results"`
			HasMore bool              `json:"has_more"`
			Next    string            `json:"next_cursor"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
			return nil, err
		}

		for _, raw := range searchResp.Results {
			var obj map[string]interface{}
			if err := json.Unmarshal(raw, &obj); err != nil {
				continue
			}

			switch obj["object"] {
			case "data_source":
				var ds struct {
					ID    string               `json:"id"`
					Title []notionapi.RichText `json:"title"`
					URL   string               `json:"url"`
				}
				if err := json.Unmarshal(raw, &ds); err == nil {
					databases = append(databases, notionapi.Database{
						ID:    notionapi.ObjectID(ds.ID),
						Title: ds.Title,
						URL:   ds.URL,
					})
				}
			case "database":
				var db notionapi.Database
				if err := json.Unmarshal(raw, &db); err == nil {
					databases = append(databases, db)
				}
			}
		}

		if !searchResp.HasMore {
			break
		}
		cursor = notionapi.Cursor(searchResp.Next)
	}

	return databases, nil
}

// GetDatabase fetches details of a specific database (or data source).
func (c *Client) GetDatabase(ctx context.Context, dbID string) (*notionapi.Database, error) {
	dbID = formatID(dbID)
	httpClient := &http.Client{Timeout: 30 * time.Second}

	// Helper to fetch a data source
	fetchDataSource := func(id string) (*notionapi.Database, error) {
		url := fmt.Sprintf("https://api.notion.com/v1/data_sources/%s", id)
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

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("status %d", resp.StatusCode)
		}

		var ds struct {
			ID         string                    `json:"id"`
			Title      []notionapi.RichText      `json:"title"`
			URL        string                    `json:"url"`
			Properties notionapi.PropertyConfigs `json:"properties"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ds); err != nil {
			return nil, err
		}

		return &notionapi.Database{
			ID:         notionapi.ObjectID(ds.ID),
			Title:      ds.Title,
			URL:        ds.URL,
			Properties: ds.Properties,
		}, nil
	}

	// 1. Try fetching as a data_source first
	if ds, err := fetchDataSource(dbID); err == nil {
		return ds, nil
	}

	// 2. If it failed, it might be a database_id.
	// Fetch database metadata to get its data_source IDs.
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s", dbID)
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
			ID          string `json:"id"`
			URL         string `json:"url"`
			DataSources []struct {
				ID string `json:"id"`
			} `json:"data_sources"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&dbMetadata); err == nil && len(dbMetadata.DataSources) > 0 {
			// Fetch the first data source to get properties
			ds, err := fetchDataSource(dbMetadata.DataSources[0].ID)
			if err == nil {
				// Use the database's own URL if preferred
				ds.URL = dbMetadata.URL
				return ds, nil
			}
		}
	}

	// 3. Fallback to the library's Get method (might not have properties)
	res, err := c.Database.Get(ctx, notionapi.DatabaseID(dbID))
	if err != nil {
		return nil, fmt.Errorf("getting database/data_source %s: %w", dbID, err)
	}
	return res, nil
}
