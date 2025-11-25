package util

import (
    "net/http"
    "net/http/httptest"
    "testing"
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
