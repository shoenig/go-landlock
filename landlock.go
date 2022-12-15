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
type Safety byte

const (
	// Mandatory mode will return an error on failure, including on
	// systems where landlock is not supported.
	Mandatory Safety = iota

	// OnlySupported will return an error on failure if running
	// on a supported operating system, or no error otherwise
	OnlySupported

	// Try mode will continue with no error on failure.
	Try
)

// A Locker is an interface over the Kernel landlock LSM feature.
type Locker interface {
	fmt.Stringer
	Lock(s Safety) error
}
