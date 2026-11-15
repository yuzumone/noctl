package notion

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ExportPageProperties retrieves and converts complete page properties to plain values.
func (c *Client) ExportPageProperties(ctx context.Context, id string) (map[string]any, error) {
	properties, _, err := c.exportPageProperties(ctx, id)
	return properties, err
}

// ExportPageReference retrieves properties and a title without depending on its property name.
func (c *Client) ExportPageReference(ctx context.Context, id string) (map[string]any, string, error) {
	return c.exportPageProperties(ctx, id)
}

func (c *Client) exportPageProperties(ctx context.Context, id string) (map[string]any, string, error) {
	title := ""
	var page struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	path := "/pages/" + url.PathEscape(formatID(id))
	if err := c.doNotionJSON(ctx, http.MethodGet, path, nil, &page, "retrieve export properties"); err != nil {
		return nil, "", err
	}
	result := make(map[string]any, len(page.Properties))
	names := make([]string, 0, len(page.Properties))
	for name := range page.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := page.Properties[name]
		kind, _ := p["type"].(string)
		switch kind {
		case "title", "rich_text", "people", "relation", "rollup":
			propertyID, _ := p["id"].(string)
			if propertyID == "" {
				return nil, "", fmt.Errorf("property %q has no ID", name)
			}
			// Property IDs returned by Notion may already be URL-encoded.
			decoded, err := url.PathUnescape(propertyID)
			if err != nil {
				return nil, "", err
			}
			endpoint := path + "/properties/" + url.PathEscape(decoded)
			values := []any{}
			cursor := ""
			seen := map[string]bool{}
			for {
				query := "?page_size=100"
				if cursor != "" {
					query += "&start_cursor=" + url.QueryEscape(cursor)
				}
				var response struct {
					Object       string           `json:"object"`
					Results      []map[string]any `json:"results"`
					HasMore      bool             `json:"has_more"`
					NextCursor   string           `json:"next_cursor"`
					PropertyItem map[string]any   `json:"property_item"`
					Type         string           `json:"type"`
					Rollup       any              `json:"rollup"`
				}
				if err := c.doNotionJSON(ctx, http.MethodGet, endpoint+query, nil, &response, "retrieve property "+name); err != nil {
					return nil, "", err
				}
				if response.Object != "list" {
					if kind != "rollup" || response.Type != "rollup" {
						return nil, "", fmt.Errorf("unexpected response for property %q", name)
					}
					p[kind] = response.Rollup
					break
				}
				for _, item := range response.Results {
					if kind == "rollup" {
						// Rollup results flatten text, people, and relations into
						// individual property items rather than page-value arrays.
						itemKind, _ := item["type"].(string)
						switch itemKind {
						case "title", "rich_text", "people", "relation":
							if _, ok := item[itemKind].(map[string]any); ok {
								item[itemKind] = []any{item[itemKind]}
							}
						}
						values = append(values, item)
					} else {
						values = append(values, item[kind])
					}
				}
				if !response.HasMore {
					if kind == "rollup" {
						rollup, _ := response.PropertyItem["rollup"].(map[string]any)
						if rollup == nil {
							return nil, "", fmt.Errorf("incomplete rollup %q", name)
						}
						if rollup["type"] == "array" {
							rollup["array"] = values
						}
						p[kind] = rollup
					} else {
						p[kind] = values
					}
					break
				}
				cursor = response.NextCursor
				if cursor == "" || seen[cursor] {
					return nil, "", fmt.Errorf("incomplete pagination for property %q", name)
				}
				seen[cursor] = true
			}
		}
		value, err := propertyValue(p)
		if err != nil {
			return nil, "", fmt.Errorf("property %q: %w", name, err)
		}
		result[name] = value
		if kind == "title" {
			title, _ = value.(string)
		}
	}
	return result, title, nil
}

func propertyValue(p map[string]any) (any, error) {
	kind, _ := p["type"].(string)
	value, exists := p[kind]
	if !exists {
		return nil, fmt.Errorf("missing value for type %q", kind)
	}
	switch kind {
	case "number", "checkbox", "url", "email", "phone_number", "created_time", "last_edited_time", "string", "boolean":
		return value, nil
	case "title", "rich_text":
		var out strings.Builder
		items, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid %s", kind)
		}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid rich text")
			}
			s, ok := m["plain_text"].(string)
			if !ok {
				return nil, fmt.Errorf("missing rich text plain_text")
			}
			out.WriteString(s)
		}
		return out.String(), nil
	case "select", "status":
		if value == nil {
			return nil, nil
		}
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid %s", kind)
		}
		return m["name"], nil
	case "multi_select", "people", "relation", "files":
		items, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid %s", kind)
		}
		out := []any{}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid %s item", kind)
			}
			switch kind {
			case "multi_select":
				out = append(out, m["name"])
			case "people", "relation":
				out = append(out, m["id"])
			case "files":
				typ, _ := m["type"].(string)
				file, ok := m[typ].(map[string]any)
				if !ok || file["url"] == nil {
					return nil, fmt.Errorf("file has no URL")
				}
				out = append(out, map[string]any{"name": m["name"], "url": file["url"]})
			}
		}
		return out, nil
	case "date":
		if value == nil {
			return nil, nil
		}
		m, ok := value.(map[string]any)
		if !ok || m["start"] == nil {
			return nil, fmt.Errorf("invalid date")
		}
		if m["end"] == nil && m["time_zone"] == nil {
			return m["start"], nil
		}
		out := map[string]any{"start": m["start"]}
		for _, k := range []string{"end", "time_zone"} {
			if m[k] != nil {
				out[k] = m[k]
			}
		}
		return out, nil
	case "created_by", "last_edited_by":
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid user")
		}
		return m["id"], nil
	case "unique_id":
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid unique_id")
		}
		return map[string]any{"prefix": m["prefix"], "number": m["number"]}, nil
	case "formula", "rollup":
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid %s", kind)
		}
		return propertyValue(m)
	case "array":
		items, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid array")
		}
		out := []any{}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid array item")
			}
			v, err := propertyValue(m)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported or incomplete property type %q", kind)
	}
}
