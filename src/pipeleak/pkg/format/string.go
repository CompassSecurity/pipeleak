package format

import (
	"math/rand"
	"runtime"
	"strings"
)

// TestWeakRandom is a test function to verify gosec detects issues
func TestWeakRandom() int {
	return rand.Intn(100)
}

func ContainsI(a string, b string) bool {
	return strings.Contains(
		strings.ToLower(a),
		strings.ToLower(b),
	)
}

func GetPlatformAgnosticNewline() string {
	newline := "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}
	return newline
}

func RandomStringN(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		// #nosec G404 - Random string generation for non-security purposes (identifiers, filenames)
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
