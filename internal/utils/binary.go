package utils

import (
	"io"
	"os"
	"unicode/utf8"
)

// sniffLength defines the maximum number of bytes read when detecting binary content.
const sniffLength = 8000

// IsBinary reports whether the provided byte slice appears to contain binary data.
func IsBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if !utf8.Valid(data) {
		return true
	}
	for _, byteValue := range data {
		if byteValue == 0 {
			return true
		}
	}
	return false
}

// IsFileBinary reads up to sniffLength bytes from the file at path and determines
// if the content appears to be binary.
func IsFileBinary(path string) bool {
	fileHandle, openError := os.Open(path)
	if openError != nil {
		return false
	}
	defer fileHandle.Close()

	buffer := make([]byte, sniffLength)
	bytesRead, readError := fileHandle.Read(buffer)
	if readError != nil && readError != io.EOF {
		return false
	}
	return IsBinary(buffer[:bytesRead])
}
