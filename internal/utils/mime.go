package utils

import (
	"io"
	"net/http"
	"os"
)

const (
	// UnknownMimeType is returned when the MIME type of a file cannot be determined.
	UnknownMimeType = EmptyString
)

// DetectMimeType determines the MIME type of the file at the provided path by
// reading up to sniffLength bytes and passing them to http.DetectContentType. It
// returns UnknownMimeType if the file cannot be read.
func DetectMimeType(filePath string) string {
	openedFile, openError := os.Open(filePath)
	if openError != nil {
		return UnknownMimeType
	}
	defer openedFile.Close()

	buffer := make([]byte, sniffLength)
	bytesRead, readError := openedFile.Read(buffer)
	if readError != nil && readError != io.EOF {
		return UnknownMimeType
	}

	return http.DetectContentType(buffer[:bytesRead])
}
