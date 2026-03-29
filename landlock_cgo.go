// Copyright (c) Seth Hoenig
// SPDX-License-Identifier: MPL-2.0

//go:build linux && cgo

package landlock

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func addProcTaskRule(fd int) error {
	procTaskPath := fmt.Sprintf("/proc/%d/task", os.Getpid())
	fd2, err := syscall.Open(procTaskPath, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	ba := beneathAttr{allowedAccess: uint64(fsReadDir)}
	ba.parentFd = fd2
	return add(fd, &ba)
}
