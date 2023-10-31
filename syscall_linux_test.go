// Copyright (c) Seth Hoenig
// SPDX-License-Identifier: MPL-2.0

//go:build linux

package landlock

import (
	"testing"
)

func TestSyscall_manual(t *testing.T) {
	caps := capabilities()
	t.Logf("caps: %x\n", caps)
}
