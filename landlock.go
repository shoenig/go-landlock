// Package landlock provides a Go library for using the
// landlock feature of the modern Linux kernel.
//
// The landlock feature of the kernel is used to isolate
// a process from accessing the filesystem except for
// blessed paths and access modes.
package landlock

import (
	"fmt"
)

// Safety indicates the enforcement behavior on systems where landlock
// does not exist or operate as expected.
//
// Enforce - return an error on failure
// Ignore - continue with no error on failure
type Safety byte

const (
	Enforce Safety = iota
	Ignore
)

// A Locker is an interface over the Kernel landlock LSM feature.
type Locker interface {
	fmt.Stringer
	Lock(s Safety) error
}
