// Package export generates static Markdown documents from Notion pages.
package export

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
	"noctl/internal/notion"
)

// Source provides the Notion data needed by the exporter.
type Source interface {
	ExportPageReference(context.Context, string) (map[string]any, string, error)
	RetrievePageMarkdown(context.Context, string) (*notion.PageMarkdown, error)
}

// Document is a completely converted page, ready to write.
type Document struct {
	Slug     string
	Markdown []byte
	Warnings []string
}

// Page exports one page without writing files or modifying Notion.
func Page(ctx context.Context, source Source, id, linkTemplate string) (*Document, error) {
	resolver, err := newResolver(ctx, source, linkTemplate)
	if err != nil {
		return nil, err
	}
	properties, title, err := source.ExportPageReference(ctx, id)
	if err != nil {
		return nil, err
	}
	slug, ok := properties["slug"].(string)
	if !ok || !validSlug(slug) {
		return nil, fmt.Errorf("slug must be a non-empty string usable as a filename")
	}
	resolver.cache[normalizeID(id)] = cachedProperties{properties: properties, title: title}
	content, err := source.RetrievePageMarkdown(ctx, id)
	if err != nil {
		return nil, err
	}
	if content == nil || content.Truncated || len(content.UnknownBlockIDs) != 0 {
		return nil, fmt.Errorf("page Markdown is incomplete")
	}
	body, err := convertMarkdown(content.Markdown, resolver)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	front, err := yaml.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("encode front matter: %w", err)
	}
	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(front)
	out.WriteString("---\n\n")
	out.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		out.WriteByte('\n')
	}
	warnings := append([]string{}, resolver.warnings...)
	for _, w := range content.Warnings {
		warnings = append(warnings, w.Code+": "+w.Message)
	}
	return &Document{Slug: slug, Markdown: out.Bytes(), Warnings: warnings}, nil
}

func validSlug(s string) bool {
	if strings.TrimSpace(s) == "" || s == "." || s == ".." || strings.HasSuffix(s, ".") || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || strings.ContainsRune(`/\<>:"|?*`, r) {
			return false
		}
	}
	return true
}

// Write saves a document atomically, refusing symlinks and accidental overwrites.
func Write(dir string, doc *Document, force bool) (string, error) {
	if !validSlug(doc.Slug) {
		return "", fmt.Errorf("invalid slug")
	}
	target := filepath.Join(dir, doc.Slug+".md")
	check := func() error {
		info, err := os.Lstat(target)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("output is not a regular file: %s", target)
		}
		if !force {
			return fmt.Errorf("output already exists (use --force): %s", target)
		}
		return nil
	}
	if err := check(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".noctl-export-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(doc.Markdown); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if force {
		if err := check(); err != nil {
			return "", err
		}
		err = os.Rename(f.Name(), target)
	} else {
		// Linking publishes the complete file and fails if another writer created target.
		err = os.Link(f.Name(), target)
	}
	if err != nil {
		return "", err
	}
	return target, nil
}
