// Copyright (c) Seth Hoenig
// SPDX-License-Identifier: MPL-2.0

//go:build linux

package landlock

import (
	"testing"

	"github.com/shoenig/test/must"
)

func Test_Available(t *testing.T) {
	a := Available()
	must.True(t, a)
}

func Test_Detect(t *testing.T) {
	v, err := Detect()
	must.NoError(t, err)

	const (
		minimum = 1 // always 1
		maximum = 4 // periodically update
	)

	must.Between(t, minimum, v, maximum)
}
