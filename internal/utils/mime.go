package utils

import (
	"io"
	"net/http"
	"os"
)

const unknownMimeType = ""

// DetectMimeType determines the MIME type of the file at the provided path by
// reading up to sniffLen bytes and passing them to http.DetectContentType. It
// returns an empty string if the file cannot be read.
func DetectMimeType(filePath string) string {
	openedFile, openError := os.Open(filePath)
	if openError != nil {
		return unknownMimeType
	}
	defer openedFile.Close()

	buffer := make([]byte, sniffLen)
	bytesRead, readError := openedFile.Read(buffer)
	if readError != nil && readError != io.EOF {
		return unknownMimeType
	}

	return http.DetectContentType(buffer[:bytesRead])
}
