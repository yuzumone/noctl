package cli

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestExportCLI(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NOCTL_NOTION_TOKEN", "test-token")
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original; viper.Reset() })
	http.DefaultTransport = transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Fatal("export must be read-only")
		}
		if req.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatal("missing token")
		}
		body := `{"properties":{"slug":{"id":"slug","type":"rich_text","rich_text":[]},"title":{"id":"title","type":"title","title":[]},"draft":{"type":"checkbox","checkbox":false}}}`
		switch {
		case strings.HasSuffix(req.URL.Path, "/markdown"):
			body = `{"markdown":"Hello\n\n<embed source=\"https://example.com\">Example</embed>","truncated":false,"unknown_block_ids":[]}`
		case strings.HasSuffix(req.URL.Path, "/properties/slug"):
			body = `{"object":"list","results":[{"rich_text":{"plain_text":"example"}}],"has_more":false}`
		case strings.HasSuffix(req.URL.Path, "/properties/title"):
			body = `{"object":"list","results":[{"title":{"plain_text":"Example"}}],"has_more":false}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	dir := t.TempDir()
	run := func(args ...string) (string, error) {
		t.Helper()
		viper.Reset()
		cmd := NewExportCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}
	out, err := run("page", "page-id", "--dryrun", "--output", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "slug: example") || !strings.Contains(out, "[Example](<https://example.com>)") {
		t.Fatal(out)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("dryrun wrote files")
	}
	out, err = run("page", "page-id", "--output", dir)
	if err != nil || out != "" {
		t.Fatalf("%q %v", out, err)
	}
	target := filepath.Join(dir, "example.md")
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = run("page", "page-id", "--output", dir); err == nil {
		t.Fatal("overwrote without force")
	}
	after, _ := os.ReadFile(target)
	if !bytes.Equal(before, after) {
		t.Fatal("modified existing file")
	}
	if _, err = run("page", "page-id", "--output", dir, "--force"); err != nil {
		t.Fatal(err)
	}
	if _, err = run("page", "page-id", "--dryrun", "--format", "other"); err == nil {
		t.Fatal("accepted invalid format")
	}
}
