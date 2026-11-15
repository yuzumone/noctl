package export

import (
	"context"
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
)

// DatabaseSource supplies database pages as well as their export content.
type DatabaseSource interface {
	Source
	QueryDatabaseAll(context.Context, string) ([]notionapi.Page, error)
}

// Database converts every database page before returning any documents.
// Duplicate filenames are rejected even when overwriting is enabled.
func Database(ctx context.Context, source DatabaseSource, id, linkTemplate string) ([]*Document, error) {
	pages, err := source.QueryDatabaseAll(ctx, id)
	if err != nil {
		return nil, err
	}
	docs := make([]*Document, 0, len(pages))
	seen := map[string]bool{}
	for _, page := range pages {
		doc, err := Page(ctx, source, string(page.ID), linkTemplate)
		if err != nil {
			return nil, fmt.Errorf("page %s: %w", page.ID, err)
		}
		key := strings.ToLower(doc.Slug)
		if seen[key] {
			return nil, fmt.Errorf("duplicate output slug %q", doc.Slug)
		}
		seen[key] = true
		docs = append(docs, doc)
	}
	return docs, nil
}
