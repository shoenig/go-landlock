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

type Locker interface {
	fmt.Stringer
	Lock(s Safety) error
}
