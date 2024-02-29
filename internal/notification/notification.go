package notification

import (
	"github.com/codegoalie/golibnotify"
	"io"
)

type Notifier interface {
	io.Closer
	Show(summary, body, icon string) error
}

type Null struct{}

func (Null) Close() error                          { return nil }
func (Null) Show(summary, body, icon string) error { return nil }

func DBus() *golibnotify.SimpleNotifier {
	return golibnotify.NewSimpleNotifier("PulseClip")
}
