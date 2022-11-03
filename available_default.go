//go:build !linux

package landlock

func Available() bool {
	return false
}
