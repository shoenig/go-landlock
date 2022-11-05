//go:build !linux

package landlock

import (
	"testing"

	"github.com/shoenig/test/must"
)

var _ Locker = (*locker)(nil)

func TestLocker_Lock_Enforce(t *testing.T) {
	l := New()
	err := l.Lock(Enforce)
	must.Error(t, err)
}

func TestLocker_Lock_Ignore(t *testing.T) {
	l := New()
	err := l.Lock(Ignore)
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
