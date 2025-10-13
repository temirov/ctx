// Package clipboard provides access to the system clipboard.
package clipboard

import (
	"github.com/atotto/clipboard"
)

// Copier copies textual data to the system clipboard.
type Copier interface {
	Copy(text string) error
}

// Service implements Copier using github.com/atotto/clipboard.
type Service struct{}

// NewService constructs a Clipboard service implementation.
func NewService() *Service {
	return &Service{}
}

// Copy writes text to the system clipboard.
func (service *Service) Copy(text string) error {
	return clipboard.WriteAll(text)
}

var _ Copier = (*Service)(nil)
