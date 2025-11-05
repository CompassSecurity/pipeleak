package artifact

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CompassSecurity/pipeleak/pkg/scanner/rules"
	"github.com/CompassSecurity/pipeleak/pkg/scanner/types"
)

func init() {
	rules.InitRules([]string{})
}

func TestDetectFileHits(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "no secrets",
			content: []byte("plain text file"),
		},
		{
			name:    "with potential secret",
			content: []byte("GITLAB_USER_ID=12345"),
		},
		{
			name:    "empty file",
			content: []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DetectFileHits(tt.content, "http://example.com/job/1", "test-job", "test.txt", "", false)
		})
	}
}

func TestReportFinding(t *testing.T) {
	finding := types.Finding{
		Pattern: types.PatternElement{
			Pattern: types.PatternPattern{
				Name:       "Test Pattern",
				Confidence: "high",
			},
		},
		Text: "secret value",
	}

	t.Run("report without archive", func(t *testing.T) {
		ReportFinding(finding, "http://example.com/job/1", "test-job", "test.txt", "")
	})

	t.Run("report with archive", func(t *testing.T) {
		ReportFinding(finding, "http://example.com/job/1", "test-job", "test.txt", "archive.zip")
	})
}

func TestHandleArchiveArtifact(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f, err := w.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("GITLAB_USER_ID=12345"))

	_ = w.Close()

	t.Run("valid zip archive", func(t *testing.T) {
		HandleArchiveArtifact("test.zip", buf.Bytes(), "http://example.com/job/1", "test-job", false)
	})

	t.Run("invalid archive data", func(t *testing.T) {
		HandleArchiveArtifact("invalid.zip", []byte("not a zip file"), "http://example.com/job/1", "test-job", false)
	})
}

func TestHandleArchiveArtifactWithDepth(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f, err := w.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("test content"))

	_ = w.Close()

	t.Run("normal depth", func(t *testing.T) {
		HandleArchiveArtifactWithDepth("test.zip", buf.Bytes(), "http://example.com/job/1", "test-job", false, 1)
	})

	t.Run("max depth exceeded", func(t *testing.T) {
		HandleArchiveArtifactWithDepth("test.zip", buf.Bytes(), "http://example.com/job/1", "test-job", false, 11)
	})

	t.Run("skipped directory - node_modules", func(t *testing.T) {
		HandleArchiveArtifactWithDepth("node_modules/test.zip", buf.Bytes(), "http://example.com/job/1", "test-job", false, 1)
	})

	t.Run("skipped directory - vendor", func(t *testing.T) {
		HandleArchiveArtifactWithDepth("vendor/test.zip", buf.Bytes(), "http://example.com/job/1", "test-job", false, 1)
	})
}

func TestHandleArchiveArtifact_NestedZip(t *testing.T) {
	innerBuf := new(bytes.Buffer)
	innerW := zip.NewWriter(innerBuf)
	innerF, _ := innerW.Create("inner.txt")
	_, _ = innerF.Write([]byte("GITLAB_USER_ID=99999"))
	_ = innerW.Close()

	outerBuf := new(bytes.Buffer)
	outerW := zip.NewWriter(outerBuf)
	outerF, _ := outerW.Create("inner.zip")
	_, _ = outerF.Write(innerBuf.Bytes())
	_ = outerW.Close()

	t.Run("nested zip archive", func(t *testing.T) {
		HandleArchiveArtifact("outer.zip", outerBuf.Bytes(), "http://example.com/job/1", "test-job", false)
	})
}

func TestDetectFileHits_RealFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "secret.txt")

	err := os.WriteFile(testFile, []byte("CI_REGISTRY_PASSWORD=mysupersecret"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	DetectFileHits(content, "http://example.com/job/1", "test-job", "secret.txt", "", false)
}
