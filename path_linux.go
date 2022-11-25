//go:build linux

package landlock

import (
	"os"
)

func (p *Path) access() rule {
	allow := rule(0)
	for _, c := range p.mode {
		switch c {
		case 'r':
			directory := fsReadFile | fsReadDir
			allow |= IfElse(p.dir, directory, fsReadFile)
		case 'w':
			allow |= fsWriteFile
		case 'x':
			allow |= fsExecute
		case 'c':
			directory := fsMakeRegular | fsMakeSocket | fsMakeFifo | fsMakeBlock | fsMakeSymlink | fsMakeDir
			allow |= IfElse(p.dir, directory, 0)
			allow |= IfElse(p.dir && version > 1, fsRefer, 0)
		}
	}
	return allow
}

var shared []*Path

var stdio []*Path

var tmp []*Path

var tty []*Path

var vminfo []*Path

var dns []*Path

var certs []*Path

func init() {
	shared = load([]*Path{
		File("/dev/null", "rw"),
		Dir("/lib", "rx"),
		Dir("/lib64", "rx"),
		Dir("/usr/lib", "rx"),
		Dir("/usr/local/lib", "rx"),
		Dir("/usr/local/lib64", "rx"),
		File("/etc/ld.so.conf", "r"),
		File("/etc/ld.so.cache", "r"),
		Dir("/etc/ld.so.conf.d", "r"),
	})

	stdio = load([]*Path{
		File("/dev/full", "rw"),
		File("/dev/zero", "r"),
		File("/dev/fd", "r"),
		File("/dev/stdin", "rw"),
		File("/dev/stdout", "rw"),
		File("/dev/urandom", "r"),
		Dir("/dev/log", "w"),
		Dir("/usr/share/locale", "r"),
		File("/proc/self/cmdline", "r"),
		File("/usr/share/zoneinfo", "r"),
		File("/usr/share/common-licenses", "r"),
		File("/proc/sys/kernel/ngroups_max", "r"),
		File("/proc/sys/kernel/cap_last_cap", "r"),
		File("/proc/sys/vm/overcommit_memory", "r"),
	})

	tty = load([]*Path{
		File("/dev/tty", "rw"),
		File("/dev/console", "rw"),
		File("/etc/terminfo", "r"),
		Dir("/usr/lib/terminfo", "r"),
		Dir("/usr/share/terminfo", "r"),
	})

	tmp = load([]*Path{
		Dir("/tmp", "rwc"),
	})

	vminfo = load([]*Path{
		File("/proc/stat", "r"),
		File("/proc/meminfo", "r"),
		File("/proc/cpuinfo", "r"),
		File("/proc/diskstats", "r"),
		File("/proc/self/maps", "r"),
		File("/proc/sys/kernel/version", "r"),
		File("/sys/devices/system/cpu", "r"),
	})

	dns = load([]*Path{
		File("/etc/hosts", "r"),
		File("/hostname", "r"),
		File("/etc/services", "r"),
		File("/etc/protocols", "r"),
		File("/etc/resolv.conf", "r"),
	})

	certs = load([]*Path{
		File("/etc/ssl/certs/ca-certificates.crt", "r"),
	})
}

func load(paths []*Path) []*Path {
	result := make([]*Path, 0, len(paths))
	for _, p := range paths {
		if _, err := os.Stat(p.path); err == nil {
			result = append(result, p)
		}
	}
	return result
}

// Shared creates a Path representing the common files and directories
// needed for dynamic shared object files.
//
// Use Shared when allowing the execution of dynamically linked binaries.
func Shared() *Path {
	return &Path{mode: modeShared}
}

// Stdio creates a Path representing the common files and directories
// needed for standard I/O operations.
func Stdio() *Path {
	return &Path{mode: modeStdio}
}

// TTY creates a path representing common files needed for terminal
// operations.
func TTY() *Path {
	return &Path{mode: modeTTY}
}

// Tmp creates a Path representing the common files and directories
// needed for reading and writing to the system tmp space.
func Tmp() *Path {
	return &Path{mode: modeTmp}
}

// VMInfo creates a Path representing the common files and directories
// needed for virtual machines and system introspection.
func VMInfo() *Path {
	return &Path{mode: modeVMInfo}
}

// DNS creates a Path representing the common files needed for DNS
// related operations.
func DNS() *Path {
	return &Path{mode: modeDNS}
}

// Certs creates a Path representing the common files needed for SSL/TLS
// certificate validation.
func Certs() *Path {
	return &Path{mode: modeCerts}
}

const (
	modeShared = "1"
	modeStdio  = "2"
	modeTTY    = "3"
	modeTmp    = "4"
	modeVMInfo = "5"
	modeDNS    = "6"
	modeCerts  = "7"
)
