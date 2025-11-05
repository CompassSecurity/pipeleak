package helper

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/CompassSecurity/pipeleak/pkg/httpclient"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		expected zerolog.Level
	}{
		{
			name:     "verbose enabled sets debug level",
			verbose:  true,
			expected: zerolog.DebugLevel,
		},
		{
			name:     "verbose disabled keeps current level",
			verbose:  false,
			expected: zerolog.Disabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zerolog.SetGlobalLevel(zerolog.Disabled)
			SetLogLevel(tt.verbose)
			if tt.verbose {
				assert.Equal(t, tt.expected, zerolog.GlobalLevel())
			}
		})
	}
}

func TestCalculateZipFileSize(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() []byte
		expected uint64
	}{
		{
			name: "valid zip file with single file",
			setup: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("test.txt")
				_, _ = f.Write([]byte("test content"))
				_ = w.Close()
				return buf.Bytes()
			},
			expected: 12,
		},
		{
			name: "valid zip file with multiple files",
			setup: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f1, _ := w.Create("test1.txt")
				_, _ = f1.Write([]byte("content1"))
				f2, _ := w.Create("test2.txt")
				_, _ = f2.Write([]byte("content2"))
				_ = w.Close()
				return buf.Bytes()
			},
			expected: 16,
		},
		{
			name: "invalid zip data returns zero",
			setup: func() []byte {
				return []byte("not a zip file")
			},
			expected: 0,
		},
		{
			name: "empty data returns zero",
			setup: func() []byte {
				return []byte{}
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.setup()
			result := CalculateZipFileSize(data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegisterGracefulShutdownHandler(t *testing.T) {
	t.Run("registers shutdown handler without panic", func(t *testing.T) {
		handler := func() {}
		assert.NotPanics(t, func() {
			RegisterGracefulShutdownHandler(handler)
		})
	})
}

func TestHeaderRoundTripper(t *testing.T) {
	tests := []struct {
		name            string
		headers         map[string]string
		requestHeaders  map[string]string
		expectedHeaders map[string]string
	}{
		{
			name: "adds default headers when not present",
			headers: map[string]string{
				"User-Agent": "test-agent",
				"Accept":     "application/json",
			},
			requestHeaders: map[string]string{},
			expectedHeaders: map[string]string{
				"User-Agent": "test-agent",
				"Accept":     "application/json",
			},
		},
		{
			name: "preserves existing request headers",
			headers: map[string]string{
				"User-Agent": "default-agent",
			},
			requestHeaders: map[string]string{
				"User-Agent": "custom-agent",
			},
			expectedHeaders: map[string]string{
				"User-Agent": "custom-agent",
			},
		},
		{
			name:            "nil headers map doesn't add headers",
			headers:         nil,
			requestHeaders:  map[string]string{},
			expectedHeaders: map[string]string{},
		},
		{
			name:            "empty headers map",
			headers:         map[string]string{},
			requestHeaders:  map[string]string{},
			expectedHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, expected := range tt.expectedHeaders {
					assert.Equal(t, expected, r.Header.Get(key))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			hrt := &httpclient.HeaderRoundTripper{
				Headers: tt.headers,
				Next:    http.DefaultTransport,
			}

			req, _ := http.NewRequest("GET", server.URL, nil)
			for key, value := range tt.requestHeaders {
				req.Header.Set(key, value)
			}

			_, err := hrt.RoundTrip(req)
			assert.NoError(t, err)
		})
	}
}

func TestGetPipeleakHTTPClient(t *testing.T) {
	tests := []struct {
		name           string
		cookieUrl      string
		cookies        []*http.Cookie
		defaultHeaders map[string]string
		validate       func(*testing.T, *http.Client)
	}{
		{
			name:           "client without cookies",
			cookieUrl:      "https://example.com",
			cookies:        []*http.Cookie{},
			defaultHeaders: map[string]string{},
			validate: func(t *testing.T, client *http.Client) {
				assert.Nil(t, client.Jar)
			},
		},
		{
			name:      "client with cookies",
			cookieUrl: "https://example.com",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
			defaultHeaders: map[string]string{},
			validate: func(t *testing.T, client *http.Client) {
				assert.NotNil(t, client.Jar)
			},
		},
		{
			name:      "client with default headers",
			cookieUrl: "https://example.com",
			cookies:   []*http.Cookie{},
			defaultHeaders: map[string]string{
				"User-Agent": "test-agent",
			},
			validate: func(t *testing.T, client *http.Client) {
				transport, ok := client.Transport.(*httpclient.HeaderRoundTripper)
				assert.True(t, ok)
				assert.NotNil(t, transport.Headers)
			},
		},
		{
			name:      "client with multiple cookies",
			cookieUrl: "https://example.com",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "token", Value: "xyz789"},
			},
			defaultHeaders: map[string]string{},
			validate: func(t *testing.T, client *http.Client) {
				assert.NotNil(t, client.Jar)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryClient := GetPipeleakHTTPClient(tt.cookieUrl, tt.cookies, tt.defaultHeaders)
			assert.NotNil(t, retryClient)
			assert.NotNil(t, retryClient.HTTPClient)
			tt.validate(t, retryClient.HTTPClient)
		})
	}
}

func TestGetPipeleakHTTPClientCheckRetry(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseError error
		expectRetry   bool
	}{
		{
			name:        "retry on 429 status",
			statusCode:  429,
			expectRetry: true,
		},
		{
			name:        "retry on 500 status",
			statusCode:  500,
			expectRetry: true,
		},
		{
			name:        "retry on 502 status",
			statusCode:  502,
			expectRetry: true,
		},
		{
			name:        "retry on 503 status",
			statusCode:  503,
			expectRetry: true,
		},
		{
			name:        "no retry on 501 status",
			statusCode:  501,
			expectRetry: false,
		},
		{
			name:        "no retry on 200 status",
			statusCode:  200,
			expectRetry: false,
		},
		{
			name:        "no retry on 404 status",
			statusCode:  404,
			expectRetry: false,
		},
		{
			name:          "retry on error",
			responseError: context.DeadlineExceeded,
			expectRetry:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := GetPipeleakHTTPClient("https://example.com", nil, nil)

			var resp *http.Response
			if tt.statusCode > 0 {
				resp = &http.Response{StatusCode: tt.statusCode}
			}

			retry, err := client.CheckRetry(context.Background(), resp, tt.responseError)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectRetry, retry)
		})
	}
}

func TestIsDirectory(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() string
		cleanup  func(string)
		expected bool
	}{
		{
			name: "existing directory returns true",
			setup: func() string {
				dir, _ := os.MkdirTemp("", "test-dir-*")
				return dir
			},
			cleanup: func(path string) {
				_ = os.RemoveAll(path)
			},
			expected: true,
		},
		{
			name: "existing file returns false",
			setup: func() string {
				f, _ := os.CreateTemp("", "test-file-*.txt")
				path := f.Name()
				_ = f.Close()
				return path
			},
			cleanup: func(path string) {
				_ = os.Remove(path)
			},
			expected: false,
		},
		{
			name: "non-existent path returns true",
			setup: func() string {
				return "/non/existent/path"
			},
			cleanup:  func(path string) {},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			defer tt.cleanup(path)
			result := IsDirectory(path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseISO8601(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected time.Time
		panics   bool
	}{
		{
			name:     "valid RFC3339 date",
			dateStr:  "2023-10-22T10:30:00Z",
			expected: time.Date(2023, 10, 22, 10, 30, 0, 0, time.UTC),
			panics:   false,
		},
		{
			name:    "valid RFC3339 with timezone",
			dateStr: "2023-10-22T10:30:00+02:00",
			expected: func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2023-10-22T10:30:00+02:00")
				return t
			}(),
			panics: false,
		},
		{
			name:     "valid RFC3339 with nanoseconds",
			dateStr:  "2023-10-22T10:30:00.123456789Z",
			expected: time.Date(2023, 10, 22, 10, 30, 0, 123456789, time.UTC),
			panics:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				assert.Panics(t, func() {
					ParseISO8601(tt.dateStr)
				})
			} else {
				result := ParseISO8601(tt.dateStr)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPrettyPrintYAML(t *testing.T) {
	tests := []struct {
		name      string
		yamlStr   string
		expectErr bool
		validate  func(*testing.T, string)
	}{
		{
			name:      "valid YAML string",
			yamlStr:   "key: value\nkey2: value2",
			expectErr: false,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "key: value")
				assert.Contains(t, result, "key2: value2")
			},
		},
		{
			name:      "nested YAML structure",
			yamlStr:   "parent:\n  child: value",
			expectErr: false,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "parent:")
				assert.Contains(t, result, "child: value")
			},
		},
		{
			name:      "invalid YAML returns error",
			yamlStr:   "key: [unclosed",
			expectErr: true,
			validate:  func(t *testing.T, result string) {},
		},
		{
			name:      "empty YAML string",
			yamlStr:   "",
			expectErr: false,
			validate: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrettyPrintYAML(tt.yamlStr)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validate(t, result)
			}
		})
	}
}

func TestContainsI(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "exact match",
			a:        "hello",
			b:        "hello",
			expected: true,
		},
		{
			name:     "case insensitive match",
			a:        "Hello World",
			b:        "world",
			expected: true,
		},
		{
			name:     "substring match",
			a:        "testing contains function",
			b:        "contains",
			expected: true,
		},
		{
			name:     "no match",
			a:        "hello",
			b:        "goodbye",
			expected: false,
		},
		{
			name:     "empty substring always matches",
			a:        "hello",
			b:        "",
			expected: true,
		},
		{
			name:     "case insensitive uppercase",
			a:        "HELLO WORLD",
			b:        "hello",
			expected: true,
		},
		{
			name:     "mixed case",
			a:        "HeLLo WoRLd",
			b:        "LLO wO",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsI(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPlatformAgnosticNewline(t *testing.T) {
	result := GetPlatformAgnosticNewline()

	if runtime.GOOS == "windows" {
		assert.Equal(t, "\r\n", result)
	} else {
		assert.Equal(t, "\n", result)
	}
}

func TestRandomStringN(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "length 5",
			length: 5,
		},
		{
			name:   "length 10",
			length: 10,
		},
		{
			name:   "length 20",
			length: 20,
		},
		{
			name:   "length 0",
			length: 0,
		},
		{
			name:   "length 1",
			length: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandomStringN(tt.length)
			assert.Equal(t, tt.length, len(result))

			for _, c := range result {
				assert.True(t, c >= 'a' && c <= 'z', "character should be lowercase letter")
			}
		})
	}

	t.Run("multiple calls produce different strings", func(t *testing.T) {
		results := make(map[string]bool)
		for i := 0; i < 100; i++ {
			results[RandomStringN(10)] = true
		}
		assert.Greater(t, len(results), 90, "should produce mostly unique strings")
	})
}

func TestExtractHTMLTitleFromB64Html(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		expected string
	}{
		{
			name:     "plain HTML with title",
			body:     []byte("<html><head><title>Test Title</title></head></html>"),
			expected: "Test Title",
		},
		{
			name:     "base64 encoded HTML with title",
			body:     []byte(base64.StdEncoding.EncodeToString([]byte("<html><head><title>Encoded Title</title></head></html>"))),
			expected: "Encoded Title",
		},
		{
			name:     "HTML without title tag",
			body:     []byte("<html><head></head><body>Content</body></html>"),
			expected: "",
		},
		{
			name:     "non-HTML content",
			body:     []byte("just plain text"),
			expected: "",
		},
		{
			name:     "empty body",
			body:     []byte(""),
			expected: "",
		},
		{
			name:     "HTML with empty title",
			body:     []byte("<html><head><title></title></head></html>"),
			expected: "",
		},
		{
			name:     "malformed HTML",
			body:     []byte("<html><title>Title</title"),
			expected: "Title</title",
		},
		{
			name:     "uppercase HTML tags",
			body:     []byte("<HTML><HEAD><TITLE>Uppercase Title</TITLE></HEAD></HTML>"),
			expected: "Uppercase Title",
		},
		{
			name:     "HTML with nested elements in title",
			body:     []byte("<html><head><title>Simple Title</title></head></html>"),
			expected: "Simple Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHTMLTitleFromB64Html(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPipeleakHTTPClientWithProxy(t *testing.T) {
	t.Run("uses HTTP_PROXY environment variable", func(t *testing.T) {
		proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer proxyServer.Close()

		_ = os.Setenv("HTTP_PROXY", proxyServer.URL)
		defer func() { _ = os.Unsetenv("HTTP_PROXY") }()

		client := GetPipeleakHTTPClient("https://example.com", nil, nil)
		assert.NotNil(t, client)
		assert.NotNil(t, client.HTTPClient.Transport)
	})
}

func BenchmarkCalculateZipFileSize(b *testing.B) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, _ := w.Create("test.txt")
	_, _ = f.Write([]byte("test content for benchmarking"))
	_ = w.Close()
	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateZipFileSize(data)
	}
}

func BenchmarkRandomStringN(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RandomStringN(10)
	}
}

func BenchmarkContainsI(b *testing.B) {
	a := "This is a Test String for Benchmarking"
	needle := "test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ContainsI(a, needle)
	}
}
