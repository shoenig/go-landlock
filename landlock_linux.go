//go:build linux

package landlock

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/hashicorp/go-set"
	"golang.org/x/sys/unix"
)

type locker struct {
	paths *set.HashSet[*Path, string]
}

// New creates a Locker that allows the given paths and permissions.
func New(paths ...*Path) Locker {
	s := set.NewHashSet[*Path, string](10)
	for _, path := range paths {
		switch path.mode {
		case modeShared:
			s.InsertAll(shared)
		case modeStdio:
			s.InsertAll(stdio)
		case modeTTY:
			s.InsertAll(tty)
		case modeTmp:
			s.InsertAll(tmp)
		case modeVMInfo:
			s.InsertAll(vminfo)
		case modeDNS:
			s.InsertAll(dns)
		case modeCerts:
			s.InsertAll(certs)
		default:
			s.Insert(path)
		}
	}
	return &locker{paths: s}
}

func (l *locker) Lock(s Safety) error {
	if !available && s != Try {
		return errors.New("landlock not available")
	}

	if err := l.lock(); err != nil && s != Try {
		return fmt.Errorf("landlock failed to lock: %w", err)
	}

	return nil
}

func (l *locker) String() string {
	return l.paths.StringFunc(func(p *Path) string {
		return fmt.Sprintf("%s:%s", p.mode, p.path)
	})
}

func (l *locker) lock() error {
	c := capabilities()
	ra := rulesetAttr{handleAccessFS: uint64(c)}

	fd, err := ruleset(&ra)
	if err != nil {
		return err
	}

	list := l.paths.Slice()
	for _, path := range list {
		if err = l.lockOne(path, fd); err != nil {
			return err
		}
	}

	if err = prctl(); err != nil {
		return err
	}

	if err = restrict(fd); err != nil {
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
	return add(fd, &ba)
}
