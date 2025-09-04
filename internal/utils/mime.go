package utils

import (
	"io"
	"net/http"
	"os"
)

// DetectMimeType returns the MIME type of the file at filePath.
// It reads up to sniffLen bytes and uses http.DetectContentType.
// If the file cannot be read, an empty string is returned.
func DetectMimeType(filePath string) string {
	fileHandle, openError := os.Open(filePath)
	if openError != nil {
		return ""
	}
	defer fileHandle.Close()

	buffer := make([]byte, sniffLen)
	bytesRead, readError := fileHandle.Read(buffer)
	if readError != nil && readError != io.EOF {
		return ""
	}

	return http.DetectContentType(buffer[:bytesRead])
}
