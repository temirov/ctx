package output

import (
	"github.com/temirov/ctx/internal/services/stream"
)

type StreamRenderer interface {
	Handle(event stream.Event) error
	Flush() error
}
