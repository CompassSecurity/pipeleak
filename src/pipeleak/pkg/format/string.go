package format

import (
"math/rand"
"runtime"
"strings"
)

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
b[i] = letterBytes[rand.Intn(len(letterBytes))]
}
return string(b)
}
