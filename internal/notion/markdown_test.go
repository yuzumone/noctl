package notion

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMarkdownAPI(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			client := &Client{token: "test", httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != method || req.URL.Path != "/v1/pages/page-id/markdown" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				if req.Header.Get("Notion-Version") != "2026-03-11" {
					t.Fatal("missing API version")
				}
				if method == http.MethodGet {
					if req.Body != nil || req.URL.Query().Get("markdown_version") != "v2" {
						t.Fatal("GET must use v2 and no body")
					}
				} else {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if string(body["markdown_version"]) != `"v2"` || string(body["type"]) != `"replace_content"` {
						t.Fatalf("unexpected body: %s", body)
					}
					if _, ok := body["allow_async"]; ok {
						t.Fatal("must stay synchronous")
					}
					var content map[string]string
					if err := json.Unmarshal(body["replace_content"], &content); err != nil {
						t.Fatal(err)
					}
					if content["new_str"] != "# Title\n\n本文" || content["allow_deleting_content"] != "" {
						t.Fatalf("unexpected content: %v", content)
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"page-id","markdown":"body","truncated":true,"unknown_block_ids":["block-id"],"warnings":[{"code":"unsupported_tag","message":"kept as text"}]}`))}, nil
			})}}
			var result *PageMarkdown
			var err error
			if method == http.MethodGet {
				result, err = client.RetrievePageMarkdown(context.Background(), "page-id")
			} else {
				result, err = client.ReplacePageMarkdown(context.Background(), "page-id", "# Title\n\n本文")
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Markdown != "body" || !result.Truncated || len(result.UnknownBlockIDs) != 1 || len(result.Warnings) != 1 {
				t.Fatalf("lost response metadata: %+v", result)
			}
		})
	}
}

func TestMarkdownAPIErrors(t *testing.T) {
	for _, status := range []int{400, 403, 404, 429, 500} {
		client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"message":"failed"}`))}, nil
		})}}
		if _, err := client.RetrievePageMarkdown(context.Background(), "id"); err == nil {
			t.Fatalf("GET accepted status %d", status)
		}
		if _, err := client.ReplacePageMarkdown(context.Background(), "id", ""); err == nil {
			t.Fatalf("PATCH accepted status %d", status)
		}
	}
}

func TestUpdatePageMarkdownChecksSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		wantSave       bool
	}{
		{"unchanged", `{"markdown":"original"}`, true},
		{"concurrent edit", `{"markdown":"changed"}`, false},
		{"truncated", `{"markdown":"original","truncated":true}`, false},
		{"unknown blocks", `{"markdown":"original","unknown_block_ids":["id"]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				body := tc.response
				if req.Method == http.MethodPatch {
					body = `{"markdown":"saved"}`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			result, err := client.UpdatePageMarkdown(context.Background(), "id", "original", "edited")
			if tc.wantSave {
				if err != nil || result.Markdown != "saved" || strings.Join(methods, ",") != "GET,PATCH" {
					t.Fatalf("save failed: %v, %v", result, err)
				}
			} else if err == nil || strings.Join(methods, ",") != "GET" {
				t.Fatalf("unsafe save: %v, %v", methods, err)
			}
		})
	}
}
