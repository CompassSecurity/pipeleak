package scanner

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestDownloadRules(t *testing.T) {
	t.Run("downloads rules when file doesn't exist", func(t *testing.T) {
		originalFileName := ruleFileName
		ruleFileName = "test-rules.yml"
		defer func() {
			_ = os.Remove(ruleFileName)
			ruleFileName = originalFileName
		}()

		_ = os.Remove(ruleFileName)
		DownloadRules()

		_, err := os.Stat(ruleFileName)
		assert.NoError(t, err, "rules file should be downloaded")
	})

	t.Run("doesn't re-download if file exists", func(t *testing.T) {
		originalFileName := ruleFileName
		ruleFileName = "test-rules-existing.yml"
		defer func() {
			_ = os.Remove(ruleFileName)
			ruleFileName = originalFileName
		}()

		testContent := "existing content"
		_ = os.WriteFile(ruleFileName, []byte(testContent), 0644)

		DownloadRules()

		content, _ := os.ReadFile(ruleFileName)
		assert.Equal(t, testContent, string(content), "should not overwrite existing file")
	})
}

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{
			name:      "invalid URL",
			url:       "http://invalid-url-that-does-not-exist-12345.com/file.txt",
			expectErr: true,
		},
		{
			name:      "empty URL",
			url:       "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(os.TempDir(), "test-download.txt")
			defer func() { _ = os.Remove(tmpFile) }()

			err := downloadFile(tt.url, tmpFile)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitRules(t *testing.T) {
	originalFileName := ruleFileName
	ruleFileName = "test-init-rules.yml"
	defer func() {
		_ = os.Remove(ruleFileName)
		ruleFileName = originalFileName
		secretsPatterns = SecretsPatterns{}
		truffelhogRules = nil
	}()

	t.Run("initializes rules without confidence filter", func(t *testing.T) {
		secretsPatterns = SecretsPatterns{}
		testYAML := `patterns:
  - pattern:
      name: Test Pattern
      regex: test_regex
      confidence: high`
		_ = os.WriteFile(ruleFileName, []byte(testYAML), 0644)

		InitRules([]string{})

		assert.Greater(t, len(secretsPatterns.Patterns), 0)
		assert.Greater(t, len(truffelhogRules), 0)
	})

	t.Run("initializes rules with confidence filter", func(t *testing.T) {
		secretsPatterns = SecretsPatterns{}
		testYAML := `patterns:
  - pattern:
      name: High Confidence Pattern
      regex: high_regex
      confidence: high
  - pattern:
      name: Low Confidence Pattern
      regex: low_regex
      confidence: low`
		_ = os.WriteFile(ruleFileName, []byte(testYAML), 0644)

		InitRules([]string{"high"})

		hasHigh := false
		hasLow := false
		for _, p := range secretsPatterns.Patterns {
			if p.Pattern.Confidence == "high" {
				hasHigh = true
			}
			if p.Pattern.Confidence == "low" {
				hasLow = true
			}
		}
		assert.True(t, hasHigh)
		assert.False(t, hasLow)
	})

	t.Run("handles empty filter that removes all rules", func(t *testing.T) {
		secretsPatterns = SecretsPatterns{}
		testYAML := `patterns:
  - pattern:
      name: High Confidence Pattern
      regex: high_regex
      confidence: high`
		_ = os.WriteFile(ruleFileName, []byte(testYAML), 0644)

		InitRules([]string{"nonexistent"})

		assert.Equal(t, 0, len(secretsPatterns.Patterns))
	})
}

func TestAppendPipeleakRules(t *testing.T) {
	tests := []struct {
		name          string
		existingRules []PatternElement
		validate      func(*testing.T, []PatternElement)
	}{
		{
			name:          "adds custom rules to empty list",
			existingRules: []PatternElement{},
			validate: func(t *testing.T, result []PatternElement) {
				assert.Greater(t, len(result), 0)
				hasGitlabRule := false
				for _, r := range result {
					if r.Pattern.Name == "Gitlab - Predefined Environment Variable" {
						hasGitlabRule = true
						break
					}
				}
				assert.True(t, hasGitlabRule)
			},
		},
		{
			name: "appends custom rules to existing rules",
			existingRules: []PatternElement{
				{Pattern: PatternPattern{Name: "Existing Rule", Regex: "test", Confidence: "high"}},
			},
			validate: func(t *testing.T, result []PatternElement) {
				assert.Greater(t, len(result), 1)
				hasExisting := false
				hasGitlab := false
				for _, r := range result {
					if r.Pattern.Name == "Existing Rule" {
						hasExisting = true
					}
					if r.Pattern.Name == "Gitlab - Predefined Environment Variable" {
						hasGitlab = true
					}
				}
				assert.True(t, hasExisting)
				assert.True(t, hasGitlab)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendPipeleakRules(tt.existingRules)
			tt.validate(t, result)
		})
	}
}

func TestDetectHits(t *testing.T) {
	secretsPatterns = SecretsPatterns{
		Patterns: []PatternElement{
			{Pattern: PatternPattern{Name: "Test Pattern", Regex: `password.*=.*[A-Za-z0-9]+`, Confidence: "high"}},
		},
	}

	tests := []struct {
		name            string
		text            []byte
		maxThreads      int
		enableVerify    bool
		expectFindings  bool
		expectedPattern string
	}{
		{
			name:            "detects password pattern",
			text:            []byte("password=secret123"),
			maxThreads:      2,
			enableVerify:    false,
			expectFindings:  true,
			expectedPattern: "Test Pattern",
		},
		{
			name:           "no match for clean text",
			text:           []byte("clean text with no secrets"),
			maxThreads:     2,
			enableVerify:   false,
			expectFindings: false,
		},
		{
			name:           "empty text",
			text:           []byte{},
			maxThreads:     2,
			enableVerify:   false,
			expectFindings: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := DetectHits(tt.text, tt.maxThreads, tt.enableVerify)
			assert.NoError(t, err)

			if tt.expectFindings {
				assert.Greater(t, len(findings), 0)
				if tt.expectedPattern != "" {
					found := false
					for _, f := range findings {
						if f.Pattern.Pattern.Name == tt.expectedPattern {
							found = true
							break
						}
					}
					assert.True(t, found, "expected pattern not found")
				}
			} else {
				assert.Equal(t, 0, len(findings))
			}
		})
	}
}

func TestDetectHitsWithTimeout(t *testing.T) {
	secretsPatterns = SecretsPatterns{
		Patterns: []PatternElement{
			{Pattern: PatternPattern{Name: "Test Pattern", Regex: `test`, Confidence: "high"}},
		},
	}

	t.Run("returns findings successfully", func(t *testing.T) {
		text := []byte("test content")
		result := DetectHitsWithTimeout(text, 2, false)

		assert.Nil(t, result.Error)
		assert.NotNil(t, result.Findings)
	})

	t.Run("handles invalid regex pattern gracefully", func(t *testing.T) {
		secretsPatterns = SecretsPatterns{
			Patterns: []PatternElement{
				{Pattern: PatternPattern{Name: "Invalid", Regex: `[invalid(regex`, Confidence: "high"}},
			},
		}
		defer func() {
			secretsPatterns = SecretsPatterns{}
		}()

		text := []byte("test content")
		result := DetectHitsWithTimeout(text, 2, false)

		assert.Nil(t, result.Error)
	})
}

func TestDeduplicateFindings(t *testing.T) {
	findingsDeduplicationList = []string{}
	defer func() {
		findingsDeduplicationList = []string{}
	}()

	tests := []struct {
		name          string
		findings      []Finding
		expectedCount int
		runTwice      bool
	}{
		{
			name: "removes duplicate findings",
			findings: []Finding{
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Regex: "test", Confidence: "high"}}, Text: "secret1"},
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Regex: "test", Confidence: "high"}}, Text: "secret1"},
			},
			expectedCount: 1,
			runTwice:      false,
		},
		{
			name: "keeps different findings",
			findings: []Finding{
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Regex: "test", Confidence: "high"}}, Text: "secret1"},
				{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Regex: "test", Confidence: "high"}}, Text: "secret2"},
			},
			expectedCount: 2,
			runTwice:      false,
		},
		{
			name:          "handles empty findings",
			findings:      []Finding{},
			expectedCount: 0,
			runTwice:      false,
		},
		{
			name: "deduplication list truncation",
			findings: func() []Finding {
				var f []Finding
				for i := 0; i < 600; i++ {
					f = append(f, Finding{
						Pattern: PatternElement{Pattern: PatternPattern{Name: "Test", Regex: "test", Confidence: "high"}},
						Text:    string(rune(i)),
					})
				}
				return f
			}(),
			expectedCount: 600,
			runTwice:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findingsDeduplicationList = []string{}
			result := deduplicateFindings(tt.findings)
			assert.Equal(t, tt.expectedCount, len(result))

			if tt.runTwice {
				result2 := deduplicateFindings(tt.findings)
				assert.Equal(t, 0, len(result2), "second run should find all duplicates")
			}
		})
	}
}

func TestDetectFileHits(t *testing.T) {
	secretsPatterns = SecretsPatterns{
		Patterns: []PatternElement{
			{Pattern: PatternPattern{Name: "Test Pattern", Regex: `password`, Confidence: "high"}},
		},
	}

	t.Run("detects hits in file content", func(t *testing.T) {
		content := []byte("password=secret")
		assert.NotPanics(t, func() {
			DetectFileHits(content, "http://example.com/job", "test-job", "test.txt", "", false)
		})
	})

	t.Run("detects hits in archived file", func(t *testing.T) {
		content := []byte("password=secret")
		assert.NotPanics(t, func() {
			DetectFileHits(content, "http://example.com/job", "test-job", "test.txt", "archive.zip", false)
		})
	})

	t.Run("handles empty content", func(t *testing.T) {
		content := []byte{}
		assert.NotPanics(t, func() {
			DetectFileHits(content, "http://example.com/job", "test-job", "test.txt", "", false)
		})
	})
}

func TestExtractHitWithSurroundingText(t *testing.T) {
	tests := []struct {
		name            string
		text            []byte
		hitIndex        []int
		additionalBytes int
		expectedContain string
	}{
		{
			name:            "extracts hit with surrounding text",
			text:            []byte("prefix password=secret suffix"),
			hitIndex:        []int{7, 22},
			additionalBytes: 5,
			expectedContain: "password=secret",
		},
		{
			name:            "handles hit at beginning",
			text:            []byte("password=secret suffix"),
			hitIndex:        []int{0, 15},
			additionalBytes: 5,
			expectedContain: "password=secret",
		},
		{
			name:            "handles hit at end",
			text:            []byte("prefix password=secret"),
			hitIndex:        []int{7, 22},
			additionalBytes: 5,
			expectedContain: "password=secret",
		},
		{
			name:            "handles additional bytes beyond text length",
			text:            []byte("short password"),
			hitIndex:        []int{6, 14},
			additionalBytes: 100,
			expectedContain: "password",
		},
		{
			name:            "zero additional bytes",
			text:            []byte("prefix password=secret suffix"),
			hitIndex:        []int{7, 22},
			additionalBytes: 0,
			expectedContain: "password=secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHitWithSurroundingText(tt.text, tt.hitIndex, tt.additionalBytes)
			assert.Contains(t, result, tt.expectedContain)
		})
	}
}

func TestCleanHitLine(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "removes newlines",
			text:     "line1\nline2\nline3",
			expected: "line1 line2 line3",
		},
		{
			name:     "removes ANSI codes",
			text:     "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "handles empty string",
			text:     "",
			expected: "",
		},
		{
			name:     "handles clean text",
			text:     "clean text",
			expected: "clean text",
		},
		{
			name:     "removes multiple newlines",
			text:     "line1\n\n\nline2",
			expected: "line1   line2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanHitLine(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleArchiveArtifact(t *testing.T) {
	secretsPatterns = SecretsPatterns{
		Patterns: []PatternElement{
			{Pattern: PatternPattern{Name: "Test Pattern", Regex: `password`, Confidence: "high"}},
		},
	}

	t.Run("handles valid zip archive", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)
		f, _ := w.Create("test.txt")
		_, _ = f.Write([]byte("password=secret"))
		_ = w.Close()

		assert.NotPanics(t, func() {
			HandleArchiveArtifact("test.zip", buf.Bytes(), "http://example.com/job", "test-job", false)
		})
	})

	t.Run("skips blocklisted directories", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)
		f, _ := w.Create("test.txt")
		_, _ = f.Write([]byte("content"))
		_ = w.Close()

		assert.NotPanics(t, func() {
			HandleArchiveArtifact("node_modules/test.zip", buf.Bytes(), "http://example.com/job", "test-job", false)
		})
	})

	t.Run("handles invalid archive data", func(t *testing.T) {
		invalidData := []byte("not an archive")
		assert.NotPanics(t, func() {
			HandleArchiveArtifact("test.zip", invalidData, "http://example.com/job", "test-job", false)
		})
	})

	t.Run("handles empty archive data", func(t *testing.T) {
		assert.NotPanics(t, func() {
			HandleArchiveArtifact("test.zip", []byte{}, "http://example.com/job", "test-job", false)
		})
	})
}

func TestHandleArchiveArtifactWithDepth(t *testing.T) {
	secretsPatterns = SecretsPatterns{
		Patterns: []PatternElement{
			{Pattern: PatternPattern{Name: "Test Pattern", Regex: `password`, Confidence: "high"}},
		},
	}

	t.Run("stops recursion at max depth", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)
		f, _ := w.Create("test.txt")
		_, _ = f.Write([]byte("password=secret"))
		_ = w.Close()

		assert.NotPanics(t, func() {
			HandleArchiveArtifactWithDepth("test.zip", buf.Bytes(), "http://example.com/job", "test-job", false, 11)
		})
	})

	t.Run("processes at depth 1", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)
		f, _ := w.Create("test.txt")
		_, _ = f.Write([]byte("password=secret"))
		_ = w.Close()

		assert.NotPanics(t, func() {
			HandleArchiveArtifactWithDepth("test.zip", buf.Bytes(), "http://example.com/job", "test-job", false, 1)
		})
	})

	t.Run("handles nested archive", func(t *testing.T) {
		innerBuf := new(bytes.Buffer)
		innerZip := zip.NewWriter(innerBuf)
		f1, _ := innerZip.Create("inner.txt")
		_, _ = f1.Write([]byte("password=secret"))
		_ = innerZip.Close()

		outerBuf := new(bytes.Buffer)
		outerZip := zip.NewWriter(outerBuf)
		f2, _ := outerZip.Create("nested.zip")
		_, _ = f2.Write(innerBuf.Bytes())
		_ = outerZip.Close()

		assert.NotPanics(t, func() {
			HandleArchiveArtifactWithDepth("outer.zip", outerBuf.Bytes(), "http://example.com/job", "test-job", false, 1)
		})
	})
}

func TestSkippableDirectoryNames(t *testing.T) {
	t.Run("contains expected blocklist entries", func(t *testing.T) {
		expectedEntries := []string{"node_modules", ".yarn", ".npm", "venv", "vendor"}
		for _, entry := range expectedEntries {
			found := false
			for _, skip := range skippableDirectoryNames {
				if skip == entry {
					found = true
					break
				}
			}
			assert.True(t, found, "expected blocklist entry %s not found", entry)
		}
	})
}

func TestDetectHitsTimeout(t *testing.T) {
	t.Run("respects timeout", func(t *testing.T) {
		secretsPatterns = SecretsPatterns{
			Patterns: []PatternElement{
				{Pattern: PatternPattern{Name: "Test Pattern", Regex: `test`, Confidence: "high"}},
			},
		}

		start := time.Now()
		text := []byte("test content")
		_, err := DetectHits(text, 1, false)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 60*time.Second, "should complete well before timeout")
	})
}

func BenchmarkDetectHits(b *testing.B) {
	secretsPatterns = SecretsPatterns{
		Patterns: []PatternElement{
			{Pattern: PatternPattern{Name: "Test Pattern", Regex: `password.*=.*[A-Za-z0-9]+`, Confidence: "high"}},
		},
	}

	text := []byte("Some text with password=secret123 and more content here")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DetectHits(text, 2, false)
	}
}

func BenchmarkCleanHitLine(b *testing.B) {
	text := "line1\nline2\n\x1b[31mred text\x1b[0m\nline3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cleanHitLine(text)
	}
}

func BenchmarkDeduplicateFindings(b *testing.B) {
	findings := []Finding{
		{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test1", Regex: "test", Confidence: "high"}}, Text: "secret1"},
		{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test2", Regex: "test", Confidence: "high"}}, Text: "secret2"},
		{Pattern: PatternElement{Pattern: PatternPattern{Name: "Test3", Regex: "test", Confidence: "high"}}, Text: "secret3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findingsDeduplicationList = []string{}
		_ = deduplicateFindings(findings)
	}
}
