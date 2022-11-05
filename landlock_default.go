//go:build !linux

package landlock

import (
	"errors"
)

var (
	// ErrNotSupported indicates Landlock is not supported on this platform
	ErrNotSupported = errors.New("landlock not supported on this platform")
)

type locker struct {
	// does nothing
}

func New(...*Path) Locker {
	return new(locker)
}

func (l *locker) Lock(s Safety) error {
	switch s {
	case Ignore:
		return nil
	default:
		return ErrNotSupported
	}
}

func (l *locker) String() string {
	return "[]"
}
