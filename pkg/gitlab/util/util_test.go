package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// TestDetermineVersion_ParsesVersion ensures the help page parsing extracts instance_version.
func TestDetermineVersion_ParsesVersion(t *testing.T) {
	// Simulate GitLab /help endpoint content containing instance_version JSON fragment
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><script>var gon={"instance_version":"16.5.1"}</script></html>`))
	}))
	defer srv.Close()

	meta := DetermineVersion(srv.URL, "")
	if meta.Version != "16.5.1" {
		t.Fatalf("expected version 16.5.1, got %s", meta.Version)
	}
	if meta.Revision != "none" || meta.Enterprise != false {
		t.Fatalf("unexpected revision/enterprise flags: %+v", meta)
	}
}

// TestDetermineVersion_FallbackWhenMissing ensures missing version returns none.
func TestDetermineVersion_FallbackWhenMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>No version here</body></html>`))
	}))
	defer srv.Close()

	meta := DetermineVersion(srv.URL, "")
	if meta.Version != "none" {
		t.Fatalf("expected version none, got %s", meta.Version)
	}
}

// TestFetchCICDYml_MissingFile ensures the function returns the correct error message when no .gitlab-ci.yml exists.
func TestFetchCICDYml_MissingFile(t *testing.T) {
	// Simulate GitLab API lint endpoint returning "Please provide content of" error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/v4/projects/") && strings.Contains(r.URL.Path, "/ci/lint") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"valid":  false,
				"errors": []string{"Please provide content of .gitlab-ci.yml"},
			}
			_ = json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = FetchCICDYml(client, 123)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "project does most certainly not have a .gitlab-ci.yml file"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("expected error to contain %q, got %q", expectedMsg, err.Error())
	}
}

// TestFetchCICDYml_ValidYAML ensures the function returns merged YAML when valid.
func TestFetchCICDYml_ValidYAML(t *testing.T) {
	expectedYAML := "stages:\n  - test\ntest-job:\n  script:\n    - echo test"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/v4/projects/") && strings.Contains(r.URL.Path, "/ci/lint") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"valid":       true,
				"errors":      []string{},
				"warnings":    []string{},
				"merged_yaml": expectedYAML,
			}
			_ = json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	yaml, err := FetchCICDYml(client, 123)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if yaml != expectedYAML {
		t.Fatalf("expected YAML %q, got %q", expectedYAML, yaml)
	}
}

// TestFetchCICDYml_OtherError ensures other validation errors are returned.
func TestFetchCICDYml_OtherError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/v4/projects/") && strings.Contains(r.URL.Path, "/ci/lint") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"valid":  false,
				"errors": []string{"syntax error on line 5"},
			}
			_ = json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = FetchCICDYml(client, 123)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "syntax error on line 5") {
		t.Fatalf("expected error to contain syntax error, got %q", err.Error())
	}
}
