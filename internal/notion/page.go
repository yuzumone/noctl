package notion

import (
	"context"
	"fmt"

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

// QueryDatabase fetches pages from a specific database.
func (c *Client) QueryDatabase(ctx context.Context, dbID string, cursor notionapi.Cursor) (*notionapi.DatabaseQueryResponse, error) {
	req := &notionapi.DatabaseQueryRequest{
		StartCursor: cursor,
		PageSize:    100,
	}

	res, err := c.Database.Query(ctx, notionapi.DatabaseID(dbID), req)
	if err != nil {
		return nil, fmt.Errorf("querying database %s: %w", dbID, err)
	}

	return res, nil
}

// CreatePage creates a new page in the specified database.
func (c *Client) CreatePage(ctx context.Context, dbID string, properties notionapi.Properties) (*notionapi.Page, error) {
	req := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: notionapi.DatabaseID(dbID),
		},
		Properties: properties,
	}

	res, err := c.Page.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating page in database %s: %w", dbID, err)
	}

	return res, nil
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
