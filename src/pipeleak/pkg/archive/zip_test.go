package archive

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateZipFileSize(t *testing.T) {
	tests := []struct {
		name         string
		createZip    func() []byte
		expectedSize uint64
		description  string
	}{
		{
			name: "empty zip",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				_ = w.Close()
				return buf.Bytes()
			},
			expectedSize: 0,
			description:  "Empty zip should have size 0",
		},
		{
			name: "zip with single small file",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				f, _ := w.Create("test.txt")
				_, _ = f.Write([]byte("Hello World"))
				_ = w.Close()
				return buf.Bytes()
			},
			expectedSize: 11, // "Hello World" is 11 bytes
			description:  "Should calculate size of single file",
		},
		{
			name: "zip with multiple files",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				f1, _ := w.Create("file1.txt")
				_, _ = f1.Write([]byte("12345"))

				f2, _ := w.Create("file2.txt")
				_, _ = f2.Write([]byte("1234567890"))

				f3, _ := w.Create("file3.txt")
				_, _ = f3.Write([]byte("abc"))

				_ = w.Close()
				return buf.Bytes()
			},
			expectedSize: 18, // 5 + 10 + 3 = 18 bytes
			description:  "Should sum sizes of all files",
		},
		{
			name: "zip with compressed file",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				// Create a file with repeating content (compresses well)
				f, _ := w.Create("repeating.txt")
				content := bytes.Repeat([]byte("A"), 1000)
				_, _ = f.Write(content)

				_ = w.Close()
				return buf.Bytes()
			},
			expectedSize: 1000, // Should return uncompressed size
			description:  "Should return uncompressed size, not compressed",
		},
		{
			name: "zip with nested directory",
			createZip: func() []byte {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				f1, _ := w.Create("dir/subdir/file.txt")
				_, _ = f1.Write([]byte("nested content"))

				_ = w.Close()
				return buf.Bytes()
			},
			expectedSize: 14, // "nested content" is 14 bytes
			description:  "Should handle nested directories",
		},
		{
			name: "invalid zip data",
			createZip: func() []byte {
				return []byte("not a zip file")
			},
			expectedSize: 0,
			description:  "Should return 0 for invalid zip",
		},
		{
			name: "empty byte array",
			createZip: func() []byte {
				return []byte{}
			},
			expectedSize: 0,
			description:  "Should return 0 for empty data",
		},
		{
			name: "nil byte array",
			createZip: func() []byte {
				return nil
			},
			expectedSize: 0,
			description:  "Should handle nil gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.createZip()
			size := CalculateZipFileSize(data)

			assert.Equal(t, tt.expectedSize, size, tt.description)
		})
	}
}

func TestCalculateZipFileSize_LargeFile(t *testing.T) {
	// Test with a larger file to ensure uint64 handling
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f, err := w.Create("large.bin")
	require.NoError(t, err)

	// Write 1MB of data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	_, err = f.Write(largeData)
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	size := CalculateZipFileSize(buf.Bytes())
	assert.Equal(t, uint64(1024*1024), size, "Should handle large files correctly")
}

func TestCalculateZipFileSize_MultipleFilesWithZeroSize(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Create multiple empty files
	for i := 0; i < 5; i++ {
		f, _ := w.Create("empty_file_" + string(rune('0'+i)) + ".txt")
		_, _ = f.Write([]byte{})
	}

	_ = w.Close()

	size := CalculateZipFileSize(buf.Bytes())
	assert.Equal(t, uint64(0), size, "Multiple empty files should total 0 bytes")
}

func TestCalculateZipFileSize_MixedSizes(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Mix of different sized files
	files := []struct {
		name    string
		content []byte
	}{
		{"tiny.txt", []byte("x")},
		{"small.txt", bytes.Repeat([]byte("y"), 100)},
		{"medium.txt", bytes.Repeat([]byte("z"), 1000)},
		{"empty.txt", []byte{}},
	}

	expectedTotal := uint64(0)
	for _, file := range files {
		f, _ := w.Create(file.name)
		_, _ = f.Write(file.content)
		expectedTotal += uint64(len(file.content))
	}

	_ = w.Close()

	size := CalculateZipFileSize(buf.Bytes())
	assert.Equal(t, expectedTotal, size, "Should correctly sum mixed file sizes")
}
