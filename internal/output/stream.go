package output

import (
	"github.com/tyemirov/ctx/internal/services/stream"
)

type StreamRenderer interface {
	Handle(event stream.Event) error
	Flush() error
}
