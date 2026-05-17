package notion

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
)

// ListDatabases fetches all databases accessible to the integration.
func (c *Client) ListDatabases(ctx context.Context) ([]notionapi.Database, error) {
	var databases []notionapi.Database
	var cursor notionapi.Cursor

	for {
		req := &notionapi.SearchRequest{
			Filter: notionapi.SearchFilter{
				Value:    "database",
				Property: "object",
			},
			StartCursor: cursor,
		}

		res, err := c.Search.Do(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("searching databases: %w", err)
		}

		for _, result := range res.Results {
			if db, ok := result.(*notionapi.Database); ok {
				databases = append(databases, *db)
			}
		}

		if !res.HasMore {
			break
		}
		cursor = res.NextCursor
	}

	return databases, nil
}

// GetDatabase fetches details of a specific database.
func (c *Client) GetDatabase(ctx context.Context, dbID string) (*notionapi.Database, error) {
	res, err := c.Database.Get(ctx, notionapi.DatabaseID(dbID))
	if err != nil {
		return nil, fmt.Errorf("getting database %s: %w", dbID, err)
	}
	return res, nil
}
