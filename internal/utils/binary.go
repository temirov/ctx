package utils

import (
	"io"
	"os"
	"unicode/utf8"
)

const sniffLen = 8000

// IsBinary reports whether the provided byte slice appears to contain binary data.
func IsBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if !utf8.Valid(data) {
		return true
	}
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// IsFileBinary reads up to sniffLen bytes from the file at path and determines
// if the content appears to be binary.
func IsFileBinary(path string) bool {
	fileHandle, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fileHandle.Close()

	buffer := make([]byte, sniffLen)
	bytesRead, err := fileHandle.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}
	return IsBinary(buffer[:bytesRead])
}
