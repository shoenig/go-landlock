//go:build !linux

package landlock

import (
	"testing"

	"github.com/shoenig/test/must"
)

var _ Locker = (*locker)(nil)

func TestLocker_Lock_Mandatory(t *testing.T) {
	l := New()
	err := l.Lock(Mandatory)
	must.Error(t, err)
}

func TestLocker_Lock_Try(t *testing.T) {
	l := New()
	err := l.Lock(Try)
	must.NoError(t, err)
}

func TestLocker_Lock_OnlySupported(t *testing.T) {
	l := New()
	err := l.Lock(OnlySupported)
	must.NoError(t, err)
}

func TestLocker_String(t *testing.T) {
	l := New()
	s := l.String()
	must.Eq(t, "[]", s)
}

func TestLocker_String_nonEmpty(t *testing.T) {
	l := New(
		Dir("/tmp", "rwc"),
	)
	s := l.String()
	must.Eq(t, "[]", s)
}
