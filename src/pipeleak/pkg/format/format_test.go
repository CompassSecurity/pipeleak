package format

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestCalculateZipFileSize(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() []byte
		expected uint64
	}{
		{
			name: "empty zip",
			setup: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				_ = w.Close()
				return buf.Bytes()
			},
			expected: 0,
		},
		{
			name: "single file",
			setup: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("test.txt")
				content := []byte("hello world")
				_, _ = f.Write(content)
				_ = w.Close()
				return buf.Bytes()
			},
			expected: 11,
		},
		{
			name: "multiple files",
			setup: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f1, _ := w.Create("test1.txt")
				_, _ = f1.Write([]byte("hello"))
				f2, _ := w.Create("test2.txt")
				_, _ = f2.Write([]byte("world"))
				_ = w.Close()
				return buf.Bytes()
			},
			expected: 10,
		},
		{
			name: "invalid zip data",
			setup: func() []byte {
				return []byte("not a zip file")
			},
			expected: 0,
		},
		{
			name: "empty data",
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
			if result != tt.expected {
				t.Errorf("CalculateZipFileSize() = %d, want %d", result, tt.expected)
			}
		})
	}
}
