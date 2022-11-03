//go:build linux

package landlock

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/hashicorp/go-set"
	"golang.org/x/sys/unix"
)

// Safety indicates the enforcement behavior on systems where landlock
// does not exist or operate as expected.
//
// Enforce - return an error on failure
// Ignore - continue with no error on failure
type Safety byte

const (
	Enforce Safety = iota
	Ignore
)

type locker struct {
	paths *set.HashSet[*Path, string]
}

func New(paths ...*Path) Locker {
	return &locker{
		paths: set.HashSetFrom[*Path, string](paths),
	}
}

func (l *locker) Lock(s Safety) error {
	if !available && s != Ignore {
		return errors.New("landlock not available")
	}

	if err := l.lock(); err != nil && s != Ignore {
		return fmt.Errorf("landlock failed to lock: %w", err)
	}

	return nil
}

func (l *locker) String() string {
	return l.paths.String(func(p *Path) string {
		return fmt.Sprintf("%s:%s", p.mode, p.path)
	})
}

func (l *locker) lock() error {
	c := capabilities()
	ra := rulesetAttr{handleAccessFS: uint64(c)}

	fd, err := ruleset(&ra, 0)
	if err != nil {
		return err
	}

	list := l.paths.List()
	for _, path := range list {
		err := l.lockOne(path, fd)
		if err != nil {
			return err
		}
	}

	if err = prctl(); err != nil {
		return err
	}

	if err = restrict(fd, 0); err != nil {
		return err
	}

	return nil
}

func (l *locker) lockOne(p *Path, fd int) error {
	allow := p.access()
	ba := beneathAttr{allowedAccess: uint64(allow)}
	fd2, err := syscall.Open(p.path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	ba.parentFd = fd2
	return add(fd, &ba, 0)
}
