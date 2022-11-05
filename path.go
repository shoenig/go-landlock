package landlock

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrImproperType indicates an improper filetype string
	ErrImproperType = errors.New("improper filetype")

	// ErrImproperMode indicates an improper mode string
	ErrImproperMode = errors.New("improper mode")

	// ErrImproperPath indicates an improper filepath string
	ErrImproperPath = errors.New("improper path")
)

type Path struct {
	mode string // any of rwxc
	path string // filepath of interest
	dir  bool   // true iff path represents a directory
}

func (p *Path) access() rule {
	allow := rule(0)
	for _, c := range p.mode {
		switch c {
		case 'r':
			directory := fsReadFile | fsReadDir
			allow |= IfElse(!p.dir, fsReadFile, directory)
		case 'w':
			allow |= fsWriteFile
		case 'x':
			allow |= fsExecute
		case 'c':
			regular := fsMakeRegular | fsMakeSocket | fsMakeFifo | fsMakeBlock | fsMakeSymlink
			allow |= IfElse(!p.dir, regular, regular|fsMakeDir)
			allow |= IfElse(version > 1, fsRefer, 0)
		}
	}
	return allow
}

var shared []*Path

// todo: stat & load
func init() {
	shared = []*Path{
		// devices
		File("/dev/null", "rw"),

		// libs
		Dir("/lib", "rx"),
		Dir("/lib64", "rx"),
		Dir("/usr/lib", "rx"),
		Dir("/usr/local/lib", "rx"),
		// Dir("/usr/local/lib64", "rx"),

		// linker
		File("/etc/ld.so.conf", "r"),
		File("/etc/ld.so.cache", "r"),
		Dir("/etc/ld.so.conf.d", "r"),
	}
}

// Equal returns true if p is equal to o in terms
// of mode and filepath.
func (p *Path) Equal(o *Path) bool {
	if p == nil || o == nil {
		return p == o
	}
	switch {
	case p.mode != o.mode:
		return false
	case p.path != o.path:
		return false
	case p.dir != o.dir:
		return false
	default:
		return true
	}
}

// Hash simply returns the path element of p.
func (p *Path) Hash() string {
	return p.path
}

func (p *Path) String() string {
	kind := IfElse(p.dir, "dir", "file")
	return fmt.Sprintf("(%s:%s:%s)", p.mode, kind, p.path)
}

// File creates a Path given the path and mode, associated with a file.
//
// File should be used with regular files, FIFOs, sockets, symlinks.
func File(path, mode string) *Path {
	return newPath(path, mode, false)
}

// Dir creates a  Path given path and mode, associated with a directory.
func Dir(path, mode string) *Path {
	return newPath(path, mode, true)
}

// Shared creates a Path representing the common files and directories
// needed for dynamic shared object files.
//
// Use Shared when allowing the execution of dynamically linked binaries.
func Shared() *Path {
	return &Path{path: "", mode: "s"}
}

func newPath(path, mode string, dir bool) *Path {
	if !IsProperPath(path) {
		panic("improper path")
	}
	if !IsProperMode(mode) {
		panic("improper mode")
	}
	return &Path{
		mode: mode,
		path: path,
		dir:  dir,
	}
}

// ParsePath parses s into a Path.
//
// s must contain 'd' or 'f' indicating whether the path represents a file
// or directory, followed by a mode string indicating the permissions of
// the path, followed by a filepath.
//
// A mode is zero or more of:
// - 'r' - enable read permission
// - 'w' - enable write permission
// - 'c' - enable create permission
// - 'x' - enable execute permission
//
// s must be in the form "[kind]:[mode]:[path]"
//
// "d:rw:$HOME" would enable reading and writing to the
// users home directory.
//
// "f:x:/bin/cat" would enable executing the /bin/cat file.
//
// It is recommended to use the File or Dir helper functions.
func ParsePath(s string) (*Path, error) {
	if s = strings.TrimSpace(s); len(s) == 0 {
		return nil, ErrImproperPath
	}
	tokens := strings.SplitN(s, ":", 3)
	if len(tokens) == 3 {
		return parsePath(tokens[0], tokens[1], tokens[2])
	}
	return nil, ErrImproperPath
}

func parsePath(filetype, mode, path string) (*Path, error) {
	switch {
	case !IsProperType(filetype):
		return nil, ErrImproperType
	case !IsProperMode(mode):
		return nil, ErrImproperMode
	case !IsProperPath(path):
		return nil, ErrImproperPath
	}
	dir := IfElse(filetype == "d", true, false)
	return &Path{
		mode: mode,
		path: path,
		dir:  dir,
	}, nil
}

func IsProperType(filetype string) bool {
	return filetype == "d" || filetype == "f"
}

// IsProperMode returns whether mode conforms to the
// "rwcx" characters of a mode string.
func IsProperMode(mode string) bool {
	if len(mode) == 0 {
		return false
	}
	for i := 0; i < len(mode); i++ {
		switch mode[i] {
		case 'r', 'w', 'c', 'x':
			continue
		default:
			return false
		}
	}
	return true
}

// IsProperPath returns whether fp conforms to a valid filepath.
func IsProperPath(path string) bool {
	return path != ""
}

func IfElse[T any](condition bool, result T, otherwise T) T {
	if condition {
		return result
	}
	return otherwise
}
