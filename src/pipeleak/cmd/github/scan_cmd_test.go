package github

import (
	"testing"
)

func TestNewScanCmd(t *testing.T) {
	cmd := NewScanCmd()

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	if cmd.Use != "scan [no options!]" {
		t.Errorf("Expected Use to be 'scan [no options!]', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	if cmd.Example == "" {
		t.Error("Expected non-empty Example")
	}

	flags := cmd.Flags()
	persistentFlags := cmd.PersistentFlags()

	if flags.Lookup("token") == nil {
		t.Error("Expected 'token' flag to exist")
	}
	if flags.Lookup("confidence") == nil {
		t.Error("Expected 'confidence' flag to exist")
	}
	if persistentFlags.Lookup("threads") == nil {
		t.Error("Expected 'threads' persistent flag to exist")
	}
	if persistentFlags.Lookup("truffleHogVerification") == nil {
		t.Error("Expected 'truffleHogVerification' persistent flag to exist")
	}
	if persistentFlags.Lookup("maxWorkflows") == nil {
		t.Error("Expected 'maxWorkflows' persistent flag to exist")
	}
	if persistentFlags.Lookup("artifacts") == nil {
		t.Error("Expected 'artifacts' persistent flag to exist")
	}
	if flags.Lookup("org") == nil {
		t.Error("Expected 'org' flag to exist")
	}
	if flags.Lookup("user") == nil {
		t.Error("Expected 'user' flag to exist")
	}
	if persistentFlags.Lookup("owned") == nil {
		t.Error("Expected 'owned' persistent flag to exist")
	}
	if persistentFlags.Lookup("public") == nil {
		t.Error("Expected 'public' persistent flag to exist")
	}
	if flags.Lookup("search") == nil {
		t.Error("Expected 'search' flag to exist")
	}
	if flags.Lookup("github") == nil {
		t.Error("Expected 'github' flag to exist")
	}
}

func TestGitHubScanOptions(t *testing.T) {
	opts := GitHubScanOptions{
		AccessToken:            "ghp_test123",
		ConfidenceFilter:       []string{"high", "verified"},
		MaxScanGoRoutines:      8,
		TruffleHogVerification: true,
		MaxWorkflows:           20,
		Organization:           "apache",
		Owned:                  false,
		User:                   "testuser",
		Public:                 true,
		SearchQuery:            "security",
		Artifacts:              true,
		GitHubURL:              "https://api.github.com",
	}

	if opts.AccessToken != "ghp_test123" {
		t.Errorf("Expected AccessToken 'ghp_test123', got %q", opts.AccessToken)
	}
	if len(opts.ConfidenceFilter) != 2 {
		t.Errorf("Expected 2 confidence filters, got %d", len(opts.ConfidenceFilter))
	}
	if opts.MaxScanGoRoutines != 8 {
		t.Errorf("Expected MaxScanGoRoutines 8, got %d", opts.MaxScanGoRoutines)
	}
	if !opts.TruffleHogVerification {
		t.Error("Expected TruffleHogVerification to be true")
	}
	if opts.MaxWorkflows != 20 {
		t.Errorf("Expected MaxWorkflows 20, got %d", opts.MaxWorkflows)
	}
	if opts.Organization != "apache" {
		t.Errorf("Expected Organization 'apache', got %q", opts.Organization)
	}
	if opts.Owned {
		t.Error("Expected Owned to be false")
	}
	if opts.User != "testuser" {
		t.Errorf("Expected User 'testuser', got %q", opts.User)
	}
	if !opts.Public {
		t.Error("Expected Public to be true")
	}
	if opts.SearchQuery != "security" {
		t.Errorf("Expected SearchQuery 'security', got %q", opts.SearchQuery)
	}
	if !opts.Artifacts {
		t.Error("Expected Artifacts to be true")
	}
	if opts.GitHubURL != "https://api.github.com" {
		t.Errorf("Expected GitHubURL 'https://api.github.com', got %q", opts.GitHubURL)
	}
}
