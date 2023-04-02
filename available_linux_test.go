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
	must.Between(t, 1, v, 3)
}
