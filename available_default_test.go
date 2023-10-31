// Copyright (c) Seth Hoenig
// SPDX-License-Identifier: MPL-2.0

//go:build !linux

package landlock

import (
	"testing"

	"github.com/shoenig/test/must"
)

func Test_Available(t *testing.T) {
	a := Available()
	must.False(t, a)
}

func Test_Detect(t *testing.T) {
	v, err := Detect()
	must.Error(t, err)
	must.Zero(t, v)
}
