package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeaderRoundTripper_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.Header.Get("Custom-Header")))
	}))
	defer server.Close()

	tests := []struct {
		name          string
		headers       map[string]string
		requestHeader map[string]string
		wantHeader    string
	}{
		{
			name:          "add default header when not present",
			headers:       map[string]string{"Custom-Header": "default-value"},
			requestHeader: map[string]string{},
			wantHeader:    "default-value",
		},
		{
			name:          "preserve existing request header",
			headers:       map[string]string{"Custom-Header": "default-value"},
			requestHeader: map[string]string{"Custom-Header": "request-value"},
			wantHeader:    "request-value",
		},
		{
			name:          "nil headers map",
			headers:       nil,
			requestHeader: map[string]string{},
			wantHeader:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hrt := &HeaderRoundTripper{
				Headers: tt.headers,
				Next:    http.DefaultTransport,
			}

			client := &http.Client{
				Transport: hrt,
			}

			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			for k, v := range tt.requestHeader {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != tt.wantHeader {
				t.Errorf("Expected header value %q, got %q", tt.wantHeader, string(body))
			}
		})
	}
}

func TestGetPipeleekHTTPClient(t *testing.T) {
	t.Run("client without cookies", func(t *testing.T) {
		client := GetPipeleekHTTPClient("", nil, nil)
		if client == nil {
			t.Fatal("Expected non-nil client")
			return
		}
		if client.Logger != nil {
			t.Error("Expected logger to be nil")
		}
	})

	t.Run("client with default headers", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "test-agent",
		}
		client := GetPipeleekHTTPClient("", nil, headers)
		if client == nil {
			t.Fatal("Expected non-nil client")
			return
		}

		hrt, ok := client.HTTPClient.Transport.(*HeaderRoundTripper)
		if !ok {
			t.Fatal("Expected HeaderRoundTripper transport")
		}

		if hrt.Headers["User-Agent"] != "test-agent" {
			t.Errorf("Expected User-Agent header to be 'test-agent', got %q", hrt.Headers["User-Agent"])
		}
	})

	t.Run("client with cookies", func(t *testing.T) {
		cookies := []*http.Cookie{
			{Name: "session", Value: "abc123"},
		}
		client := GetPipeleekHTTPClient("http://example.com", cookies, nil)
		if client == nil {
			t.Fatal("Expected non-nil client")
			return
		}
		if client.HTTPClient.Jar == nil {
			t.Error("Expected cookie jar to be set")
		}
	})

	t.Run("check retry function", func(t *testing.T) {
		client := GetPipeleekHTTPClient("", nil, nil)

		shouldRetry, _ := client.CheckRetry(nil, &http.Response{StatusCode: 429}, nil)
		if !shouldRetry {
			t.Error("Expected to retry on 429 status")
		}

		shouldRetry, _ = client.CheckRetry(nil, &http.Response{StatusCode: 500}, nil)
		if !shouldRetry {
			t.Error("Expected to retry on 500 status")
		}

		shouldRetry, _ = client.CheckRetry(nil, &http.Response{StatusCode: 501}, nil)
		if shouldRetry {
			t.Error("Expected NOT to retry on 501 status")
		}

		shouldRetry, _ = client.CheckRetry(nil, &http.Response{StatusCode: 200}, nil)
		if shouldRetry {
			t.Error("Expected NOT to retry on 200 status")
		}

		shouldRetry, _ = client.CheckRetry(nil, nil, nil)
		if shouldRetry {
			t.Error("Expected NOT to retry with nil response")
		}

		shouldRetry, _ = client.CheckRetry(nil, nil, http.ErrServerClosed)
		if !shouldRetry {
			t.Error("Expected to retry on error")
		}
	})
}
