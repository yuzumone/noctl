package notion

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDoNotionJSONUsesConfiguredHTTPClient(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotVersion string
	var gotContentType string

	client := &Client{
		token: "secret",
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotPath = req.URL.Path
			gotAuth = req.Header.Get("Authorization")
			gotVersion = req.Header.Get("Notion-Version")
			gotContentType = req.Header.Get("Content-Type")

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})},
	}

	var response struct {
		OK bool `json:"ok"`
	}
	err := client.doNotionJSON(context.Background(), http.MethodPost, "/test", map[string]string{"name": "value"}, &response, "test")
	if err != nil {
		t.Fatalf("doNotionJSON returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/v1/test" {
		t.Fatalf("path = %q, want %q", gotPath, "/v1/test")
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("authorization header = %q, want %q", gotAuth, "Bearer secret")
	}
	if gotVersion != "2026-03-11" {
		t.Fatalf("notion version header = %q, want %q", gotVersion, "2026-03-11")
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type header = %q, want %q", gotContentType, "application/json")
	}
	if !response.OK {
		t.Fatal("response.OK = false, want true")
	}
}

func TestRetryTransportRetriesWithRequestBody(t *testing.T) {
	var bodies []string
	var delays []time.Duration

	transport := &retryTransport{
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			bodies = append(bodies, string(body))

			status := http.StatusOK
			if len(bodies) == 1 {
				status = http.StatusInternalServerError
			}

			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
		sleep: func(delay time.Duration) {
			delays = append(delays, delay)
		},
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.notion.com/v1/test", bytes.NewReader([]byte("payload")))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	wantBodies := []string{"payload", "payload"}
	if len(bodies) != len(wantBodies) {
		t.Fatalf("bodies = %v, want %v", bodies, wantBodies)
	}
	for i := range bodies {
		if bodies[i] != wantBodies[i] {
			t.Fatalf("bodies = %v, want %v", bodies, wantBodies)
		}
	}

	if len(delays) != 1 || delays[0] != 500*time.Millisecond {
		t.Fatalf("delays = %v, want [500ms]", delays)
	}
}
