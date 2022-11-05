//go:build !linux

package landlock

// Detect returns ErrNotSupported on non-Linux platforms.
func Detect() (int, error) {
	return 0, ErrNotSupported
}

// Available returns false on non-Linux platforms.
func Available() bool {
	return false
}
