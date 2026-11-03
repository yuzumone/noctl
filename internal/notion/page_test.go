package notion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type queryStep struct {
	method, path string
	status       int
	body         string
	cursor       string
	err          error
}

func queryClient(t *testing.T, steps []queryStep) *Client {
	t.Helper()
	index := 0
	t.Cleanup(func() {
		if index != len(steps) {
			t.Errorf("made %d requests, want %d", index, len(steps))
		}
	})
	return &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if index >= len(steps) {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		step := steps[index]
		index++
		if req.Method != step.method || req.URL.Path != step.path {
			t.Fatalf("request = %s %s, want %s %s", req.Method, req.URL.Path, step.method, step.path)
		}
		if req.Method == http.MethodPost {
			var body struct {
				PageSize int    `json:"page_size"`
				Cursor   string `json:"start_cursor"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.PageSize != 100 || body.Cursor != step.cursor {
				t.Fatalf("unexpected pagination: %+v, want cursor %q", body, step.cursor)
			}
		}
		if step.err != nil {
			return nil, step.err
		}
		return &http.Response{StatusCode: step.status, Body: io.NopCloser(strings.NewReader(step.body))}, nil
	})}}
}

func databaseResolutionSteps(last queryStep) []queryStep {
	return []queryStep{
		{method: "POST", path: "/v1/data_sources/database/query", status: 404, body: `{"message":"not a data source"}`},
		{method: "GET", path: "/v1/databases/database", status: 200, body: `{"data_sources":[{"id":"source"}]}`},
		last,
	}
}

func TestQueryDatabaseResolvedSourceErrors(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client := queryClient(t, databaseResolutionSteps(queryStep{
				method: "POST", path: "/v1/data_sources/source/query", status: status, body: `{"message":"source query failed"}`,
			}))
			res, err := client.QueryDatabase(context.Background(), "database", "")
			if res != nil || err == nil {
				t.Fatalf("QueryDatabase = (%v, %v), want nil response and error", res, err)
			}
			for _, detail := range []string{"source", fmt.Sprint(status), "source query failed"} {
				if !strings.Contains(err.Error(), detail) {
					t.Errorf("error %q does not include %q", err, detail)
				}
			}
		})
	}
}

func TestQueryDatabaseAllPropagatesResolvedSourceError(t *testing.T) {
	client := queryClient(t, databaseResolutionSteps(queryStep{
		method: "POST", path: "/v1/data_sources/source/query", status: 500, body: `{"message":"query failed"}`,
	}))
	pages, err := client.QueryDatabaseAll(context.Background(), "database")
	if pages != nil || err == nil {
		t.Fatalf("QueryDatabaseAll = (%v, %v), want error without panic", pages, err)
	}
}

func TestQueryDatabaseStopsOnNonResolutionErrors(t *testing.T) {
	for _, status := range []int{401, 403, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client := queryClient(t, []queryStep{{method: "POST", path: "/v1/data_sources/source/query", status: status, body: `{"message":"failed"}`}})
			if _, err := client.QueryDatabase(context.Background(), "source", ""); err == nil {
				t.Fatal("expected error without trying another endpoint")
			}
		})
	}
	for _, transportErr := range []error{context.Canceled, errors.New("connection failed")} {
		client := queryClient(t, []queryStep{{method: "POST", path: "/v1/data_sources/source/query", err: transportErr}})
		if _, err := client.QueryDatabase(context.Background(), "source", ""); !errors.Is(err, transportErr) {
			t.Fatalf("lost transport error: %v", err)
		}
	}
	client := queryClient(t, []queryStep{{method: "POST", path: "/v1/data_sources/source/query", status: 200, body: "invalid JSON"}})
	if _, err := client.QueryDatabase(context.Background(), "source", ""); err == nil {
		t.Fatal("expected decoding error without fallback")
	}
}

func TestQueryDatabaseResolutionAndPagination(t *testing.T) {
	for _, status := range []int{400, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			steps := databaseResolutionSteps(queryStep{method: "POST", path: "/v1/data_sources/source/query", status: 200, body: `{"results":[{"id":"page"}]}`})
			steps[0].status = status
			client := queryClient(t, steps)
			res, err := client.QueryDatabase(context.Background(), "database", "")
			if err != nil || res == nil || len(res.Results) != 1 {
				t.Fatalf("resolution failed: %v, %v", res, err)
			}
		})
	}
	client := queryClient(t, []queryStep{
		{method: "POST", path: "/v1/data_sources/source/query", status: 200, body: `{"results":[{"id":"first"}],"has_more":true,"next_cursor":"next"}`},
		{method: "POST", path: "/v1/data_sources/source/query", status: 200, body: `{"results":[{"id":"second"}],"has_more":false}`, cursor: "next"},
	})
	pages, err := client.QueryDatabaseAll(context.Background(), "source")
	if err != nil || len(pages) != 2 || pages[1].ID != "second" {
		t.Fatalf("pagination failed: %v, %v", pages, err)
	}
}

func TestQueryDatabaseMetadataErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		err        error
	}{
		{name: "forbidden", status: 403, body: `{"message":"access denied"}`},
		{name: "rate limited", status: 429, body: `{"message":"rate limited"}`},
		{name: "server error", status: 500, body: `{"message":"server error"}`},
		{name: "invalid JSON", status: 200, body: "invalid JSON"},
		{name: "canceled", err: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := queryClient(t, []queryStep{
				{method: "POST", path: "/v1/data_sources/database/query", status: 404, body: "{}"},
				{method: "GET", path: "/v1/databases/database", status: tc.status, body: tc.body, err: tc.err},
			})
			res, err := client.QueryDatabase(context.Background(), "database", "")
			if res != nil || err == nil {
				t.Fatalf("QueryDatabase = (%v, %v), want metadata error without fallback", res, err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) {
				t.Fatalf("lost metadata transport error: %v", err)
			}
		})
	}
}

func TestQueryDatabaseLegacyFallback(t *testing.T) {
	for _, tc := range []struct {
		name           string
		metadataStatus int
		queryStatus    int
	}{
		{name: "metadata not found", metadataStatus: 404, queryStatus: 200},
		{name: "metadata bad request", metadataStatus: 400, queryStatus: 200},
		{name: "no data sources", metadataStatus: 200, queryStatus: 200},
		{name: "query failed", metadataStatus: 404, queryStatus: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := queryClient(t, []queryStep{
				{method: "POST", path: "/v1/data_sources/database/query", status: 404, body: "{}"},
				{method: "GET", path: "/v1/databases/database", status: tc.metadataStatus, body: "{}"},
				{method: "POST", path: "/v1/databases/database/query", status: tc.queryStatus, body: `{"results":[{"id":"legacy"}]}`},
			})
			res, err := client.QueryDatabase(context.Background(), "database", "")
			if tc.queryStatus != http.StatusOK {
				if res != nil || err == nil {
					t.Fatalf("lost legacy query failure: %v, %v", res, err)
				}
			} else if err != nil || res == nil || len(res.Results) != 1 || res.Results[0].ID != "legacy" {
				t.Fatalf("legacy fallback failed: %v, %v", res, err)
			}
		})
	}
}
