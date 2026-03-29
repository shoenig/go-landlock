// Copyright (c) Seth Hoenig
// SPDX-License-Identifier: MPL-2.0

//go:build linux && !cgo

package landlock

func addProcTaskRule(fd int) error {
	return nil
}
