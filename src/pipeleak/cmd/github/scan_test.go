package github

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteHighestXKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[int64]struct{}
		nrKeys   int
		expected map[int64]struct{}
	}{
		{
			name: "delete highest 2 keys from 5",
			input: map[int64]struct{}{
				1: {}, 2: {}, 3: {}, 4: {}, 5: {},
			},
			nrKeys: 2,
			expected: map[int64]struct{}{
				1: {}, 2: {}, 3: {},
			},
		},
		{
			name: "delete all keys when nrKeys equals map size",
			input: map[int64]struct{}{
				10: {}, 20: {}, 30: {},
			},
			nrKeys:   3,
			expected: map[int64]struct{}{},
		},
		{
			name: "return empty map when nrKeys exceeds map size",
			input: map[int64]struct{}{
				1: {}, 2: {},
			},
			nrKeys:   5,
			expected: map[int64]struct{}{},
		},
		{
			name: "delete nothing when nrKeys is 0",
			input: map[int64]struct{}{
				100: {}, 200: {}, 300: {},
			},
			nrKeys: 0,
			expected: map[int64]struct{}{
				100: {}, 200: {}, 300: {},
			},
		},
		{
			name:     "handle empty map",
			input:    map[int64]struct{}{},
			nrKeys:   1,
			expected: map[int64]struct{}{},
		},
		{
			name: "delete single highest key",
			input: map[int64]struct{}{
				5: {}, 10: {}, 15: {}, 20: {},
			},
			nrKeys: 1,
			expected: map[int64]struct{}{
				5: {}, 10: {}, 15: {},
			},
		},
		{
			name: "handle negative keys correctly",
			input: map[int64]struct{}{
				-10: {}, -5: {}, 0: {}, 5: {}, 10: {},
			},
			nrKeys: 2,
			expected: map[int64]struct{}{
				-10: {}, -5: {}, 0: {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deleteHighestXKeys(tt.input, tt.nrKeys)
			assert.Equal(t, tt.expected, result, "Result map should match expected")
		})
	}
}

func TestReadZipFile(t *testing.T) {
	t.Run("successfully reads file from zip", func(t *testing.T) {
		// Create a zip file in memory
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)

		testContent := []byte("test file content")
		f, err := w.Create("testfile.txt")
		require.NoError(t, err)
		_, err = f.Write(testContent)
		require.NoError(t, err)

		err = w.Close()
		require.NoError(t, err)

		// Read the zip
		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)
		require.Len(t, zipReader.File, 1)

		// Test readZipFile
		content, err := readZipFile(zipReader.File[0])
		assert.NoError(t, err)
		assert.Equal(t, testContent, content)
	})

	t.Run("reads empty file from zip", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)

		f, err := w.Create("emptyfile.txt")
		require.NoError(t, err)
		_, err = f.Write([]byte{})
		require.NoError(t, err)

		err = w.Close()
		require.NoError(t, err)

		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)

		content, err := readZipFile(zipReader.File[0])
		assert.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("reads large file from zip", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)

		// Create a large file (1MB)
		largeContent := bytes.Repeat([]byte("a"), 1024*1024)
		f, err := w.Create("largefile.txt")
		require.NoError(t, err)
		_, err = f.Write(largeContent)
		require.NoError(t, err)

		err = w.Close()
		require.NoError(t, err)

		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)

		content, err := readZipFile(zipReader.File[0])
		assert.NoError(t, err)
		assert.Equal(t, len(largeContent), len(content))
		assert.Equal(t, largeContent, content)
	})

	t.Run("handles binary file from zip", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)

		binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		f, err := w.Create("binaryfile.bin")
		require.NoError(t, err)
		_, err = f.Write(binaryContent)
		require.NoError(t, err)

		err = w.Close()
		require.NoError(t, err)

		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)

		content, err := readZipFile(zipReader.File[0])
		assert.NoError(t, err)
		assert.Equal(t, binaryContent, content)
	})
}
