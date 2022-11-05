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
