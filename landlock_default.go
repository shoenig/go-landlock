//go:build !linux

package landlock

type locker struct {
	// does nothing
}

func (l *locker) Lock() {
	// does nothing
}

func (l *locker) String() string {
	return "[]"
}
