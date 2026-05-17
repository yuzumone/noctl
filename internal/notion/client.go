package notion

import (
	"net/http"
	"time"

	"github.com/jomei/notionapi"
)

// Client is a wrapper around notionapi.Client.
type Client struct {
	*notionapi.Client
}

// retryTransport is a http.RoundTripper that retries on 429 and 5xx errors.
type retryTransport struct {
	base http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < 3; i++ {
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate limited, wait and retry
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		if resp.StatusCode >= 500 {
			// Server error, wait and retry
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}

		return resp, nil
	}

	return resp, err
}

// NewClient creates a new Notion client with retry logic.
func NewClient(token string) *Client {
	httpClient := &http.Client{
		Transport: &retryTransport{
			base: http.DefaultTransport,
		},
		Timeout: 30 * time.Second,
	}

	return &Client{
		Client: notionapi.NewClient(notionapi.Token(token), notionapi.WithHTTPClient(httpClient)),
	}
}
