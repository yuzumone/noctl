package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestExportPropertyValues(t *testing.T) {
	tests := []struct {
		name, raw string
		want      any
		fail      bool
	}{
		{"text", `{"type":"rich_text","rich_text":[{"plain_text":"hello\n"},{"plain_text":"world"}]}`, "hello\nworld", false},
		{"empty text", `{"type":"title","title":[]}`, "", false},
		{"number", `{"type":"number","number":0}`, float64(0), false},
		{"null", `{"type":"number","number":null}`, nil, false},
		{"checkbox", `{"type":"checkbox","checkbox":false}`, false, false},
		{"status", `{"type":"status","status":{"name":"Ready","color":"red"}}`, "Ready", false},
		{"select", `{"type":"select","select":null}`, nil, false},
		{"tags", `{"type":"multi_select","multi_select":[{"name":"Go"},{"name":"Notion"}]}`, []any{"Go", "Notion"}, false},
		{"empty tags", `{"type":"multi_select","multi_select":[]}`, []any{}, false},
		{"date", `{"type":"date","date":{"start":"2026-11-03","end":null,"time_zone":null}}`, "2026-11-03", false},
		{"date range", `{"type":"date","date":{"start":"2026-11-03","end":"2026-11-05","time_zone":null}}`, map[string]any{"start": "2026-11-03", "end": "2026-11-05"}, false},
		{"date zone", `{"type":"date","date":{"start":"2026-11-03T09:00:00","end":null,"time_zone":"Asia/Tokyo"}}`, map[string]any{"start": "2026-11-03T09:00:00", "time_zone": "Asia/Tokyo"}, false},
		{"people", `{"type":"people","people":[{"id":"user","name":"User"}]}`, []any{"user"}, false},
		{"relation", `{"type":"relation","relation":[{"id":"page"}]}`, []any{"page"}, false},
		{"files", `{"type":"files","files":[{"name":"photo","type":"external","external":{"url":"https://img.example.com/a"}}]}`, []any{map[string]any{"name": "photo", "url": "https://img.example.com/a"}}, false},
		{"user", `{"type":"created_by","created_by":{"id":"user"}}`, "user", false},
		{"unique", `{"type":"unique_id","unique_id":{"prefix":"TASK","number":1}}`, map[string]any{"prefix": "TASK", "number": float64(1)}, false},
		{"formula", `{"type":"formula","formula":{"type":"boolean","boolean":false}}`, false, false},
		{"rollup", `{"type":"rollup","rollup":{"type":"array","array":[{"type":"number","number":5}]}}`, []any{float64(5)}, false},
		{"incomplete", `{"type":"rollup","rollup":{"type":"incomplete","incomplete":{}}}`, nil, true},
		{"unknown", `{"type":"future","future":"x"}`, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p map[string]any
			if err := json.Unmarshal([]byte(tt.raw), &p); err != nil {
				t.Fatal(err)
			}
			got, err := propertyValue(p)
			if (err != nil) != tt.fail {
				t.Fatalf("%v", err)
			}
			if !tt.fail && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestExportPropertyPagination(t *testing.T) {
	requests := 0
	client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != "GET" {
			t.Fatal(req.Method)
		}
		body := `{"properties":{"slug":{"id":"s%3Ax","type":"rich_text","rich_text":[{"plain_text":"truncated"}]},"related":{"id":"r","type":"relation","relation":[],"has_more":true},"total":{"id":"n","type":"rollup","rollup":{"type":"incomplete","incomplete":{}}}}}`
		switch req.URL.Path {
		case "/v1/pages/page":
		case "/v1/pages/page/properties/s:x":
			if strings.Contains(req.URL.EscapedPath(), "%253A") {
				t.Fatal("double-encoded ID")
			}
			body = `{"object":"list","results":[{"type":"rich_text","rich_text":{"plain_text":"hello"}}],"has_more":true,"next_cursor":"next"}`
			if req.URL.Query().Get("start_cursor") == "next" {
				body = `{"object":"list","results":[{"type":"rich_text","rich_text":{"plain_text":"world"}}],"has_more":false}`
			}
		case "/v1/pages/page/properties/r":
			body = `{"object":"list","results":[{"type":"relation","relation":{"id":"related-page"}}],"has_more":false}`
		case "/v1/pages/page/properties/n":
			body = `{"object":"list","results":[],"has_more":false,"property_item":{"rollup":{"type":"number","number":12}}}`
		default:
			return nil, fmt.Errorf("unexpected path %s", req.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	got, err := client.ExportPageProperties(context.Background(), "page")
	if err != nil {
		t.Fatal(err)
	}
	if got["slug"] != "helloworld" || got["total"] != float64(12) || !reflect.DeepEqual(got["related"], []any{"related-page"}) || requests != 5 {
		t.Fatalf("%#v requests %d", got, requests)
	}
}

func TestExportPropertyIncompletePagination(t *testing.T) {
	client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"properties":{"slug":{"id":"s","type":"rich_text","rich_text":[]}}}`
		if strings.Contains(req.URL.Path, "/properties/") {
			body = `{"object":"list","results":[],"has_more":true,"next_cursor":null}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	if _, err := client.ExportPageProperties(context.Background(), "page"); err == nil {
		t.Fatal("accepted incomplete property")
	}
}
