//go:build linux

package landlock

import (
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/shoenig/test/must"
)

func TestLocker_New(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		l := New()
		must.NotNil(t, l)
	})

	t.Run("full", func(t *testing.T) {
		l := New(Dir("/etc", "r"))
		must.NotNil(t, l)
	})
}

func TestLocker_String(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		l := New()
		result := l.String()
		must.EqOp(t, "[]", result)
	})

	t.Run("full", func(t *testing.T) {
		l := New(
			Dir("/home/nobody", "r"),
			Dir("/opt/bin", "x"),
			Dir("~/", "rwc"),
		)
		result := l.String()
		must.Eq(t, "[r:/home/nobody rwc:~/ x:/opt/bin]", result)
	})
}

func TestLocker_reads(t *testing.T) {
	type testCase struct {
		paths   []string
		success []string
		failure []string
	}

	try := func(tc testCase) {
		var paths []*Path
		for _, path := range tc.paths {
			p, err := ParsePath(path)
			must.NoError(t, err)
			paths = append(paths, p)
		}
		err := New(paths...).Lock(Mandatory)
		must.NoError(t, err, must.Sprint("paths", paths))

		for _, p := range tc.failure {
			_, err = os.ReadFile(p)
			must.Error(t, err)
			must.StrHasSuffix(t, "permission denied", err.Error())
		}
		for _, p := range tc.success {
			_, err = os.ReadFile(p)
			must.NoError(t, err)
		}
	}

	cases := map[string]func(){
		"none": func() {
			try(testCase{
				paths:   nil,
				success: nil,
				failure: []string{
					"tests/Labels.txt",
					"tests/fruits",
					"tests/fruits/apple.txt",
				}},
			)
		},
		"write only file": func() {
			try(testCase{
				paths: []string{"f:w:tests/Labels.txt"},
				failure: []string{
					"tests/Labels.txt",
					"tests/fruits/apple.txt",
				},
			})
		},
		"write only dir": func() {
			try(testCase{
				paths:   []string{"d:w:tests/"},
				success: nil,
				failure: []string{
					"tests/Labels.txt",
					"tests/fruits/apple.txt",
				},
			})
		},
		"read top file": func() {
			try(testCase{
				paths:   []string{"f:r:tests/Labels.txt"},
				success: []string{"tests/Labels.txt"},
				failure: []string{
					"tests/fruits/apple.txt",
					"tests/fruits/banana.txt",
					"tests/veggies/celary.txt",
					"tests/veggies/corn.txt",
					"tests/veggies/unsure/beans.txt",
				},
			})
		},
		"read top dir": func() {
			try(testCase{
				paths: []string{"d:r:tests"},
				success: []string{
					"tests/Labels.txt",
					"tests/fruits/apple.txt",
					"tests/fruits/banana.txt",
					"tests/veggies/celary.txt",
					"tests/veggies/corn.txt",
					"tests/veggies/unsure/beans.txt",
				},
			})
		},
		"read fruits dir": func() {
			try(testCase{
				paths: []string{"d:r:tests/fruits"},
				success: []string{
					"tests/fruits/apple.txt",
					"tests/fruits/banana.txt",
				},
				failure: []string{
					"tests/Labels.txt",
					"tests/veggies/celary.txt",
					"tests/veggies/corn.txt",
					"tests/veggies/unsure/beans.txt",
				},
			})
		},
		"read beans file": func() {
			try(testCase{
				paths: []string{"f:r:tests/veggies/unsure/beans.txt"},
				success: []string{
					"tests/veggies/unsure/beans.txt",
				},
				failure: []string{
					"tests/Labels.txt",
					"tests/fruits/apple.txt",
					"tests/fruits/banana.txt",
					"tests/veggies/corn.txt",
					"tests/veggies/celary.txt",
				},
			})
		},
		"read mixed file": func() {
			try(testCase{
				paths: []string{
					// "f:rw:tests/veggies/corn.txt", // should this work?
					"f:rw:tests/fruits/apple.txt",
				},
				success: []string{
					"tests/fruits/apple.txt",
				},
				failure: []string{
					"tests/Labels.txt",
					"tests/veggies/corn.txt",
					"tests/fruits/banana.txt",
					"tests/veggies/celary.txt",
				},
			})
		},
		"etc directory": func() {
			try(testCase{
				paths:   []string{"d:r:/etc"},
				success: []string{"/etc/passwd"},
				failure: []string{"/bin/sh"},
			})
		},
	}

	// if we are child process, run the assigned test case
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_reads", cases)
}

func TestLocker_writes(t *testing.T) {
	cases := map[string]func(){
		"none": func() {
			l := New() // none
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.WriteFile("/tmp/a.txt", []byte{'a'}, 0o640)
			must.Error(t, err)
		},
		"one": func() {
			f := tmpFile(t, "hello.txt", "hi")
			l := New(File(f, "w"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			w, err := os.OpenFile(f, os.O_WRONLY, 0o644)
			must.NoError(t, err)
			_, err = io.WriteString(w, "test")
			must.NoError(t, err)
			err = w.Close()
			must.NoError(t, err)
			_, err = os.ReadFile(f)
			must.Error(t, err) // no permission
		},
		"one_rw": func() {
			f := tmpFile(t, "hello.txt", "hi")
			l := New(File(f, "rw"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			w, err := os.OpenFile(f, os.O_WRONLY, 0o644)
			must.NoError(t, err)
			_, err = io.WriteString(w, "test")
			must.NoError(t, err)
			err = w.Close()
			must.NoError(t, err)
			_, err = os.ReadFile(f)
			must.NoError(t, err) // has permission
		},
		"dir_ro": func() {
			f := tmpFile(t, "hello.txt", "hi")
			l := New(Dir(filepath.Dir(f), "r"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			_, err = os.OpenFile(f, os.O_WRONLY, 0o644)
			must.Error(t, err)
		},
		"dir_rw": func() {
			f := tmpFile(t, "hello.txt", "hi")
			l := New(Dir(filepath.Dir(f), "rw"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			w, err := os.OpenFile(f, os.O_WRONLY, 0o644)
			must.NoError(t, err)
			_, err = io.WriteString(w, "test")
			must.NoError(t, err)
			err = w.Close()
			must.NoError(t, err)
			_, err = os.ReadFile(f)
			must.NoError(t, err) // has permission
		},
		"dir_w": func() {
			f := tmpFile(t, "hello.txt", "hi")
			l := New(Dir(filepath.Dir(f), "w"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			w, err := os.OpenFile(f, os.O_WRONLY, 0o644)
			must.NoError(t, err)
			_, err = io.WriteString(w, "test")
			must.NoError(t, err)
			err = w.Close()
			must.NoError(t, err)
			_, err = os.ReadFile(f)
			must.Error(t, err) // no permission
		},
	}

	// if we are child process, run the assigned test case
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_writes", cases)
}

func TestLocker_truncate(t *testing.T) {
	requiresVersion(t, 3)

	cases := map[string]func(){
		"truncate_none": func() {
			f := tmpFile(t, "hi.txt", "hello")
			l := New()
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Truncate(f, 1024)
			must.Error(t, err)
		},
		"truncate_file_rcx": func() {
			f := tmpFile(t, "hi.txt", "hello")
			l := New(File(f, "rcx"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Truncate(f, 1024)
			must.Error(t, err)
		},
		"truncate_file_w": func() {
			f := tmpFile(t, "hi.txt", "hello")
			l := New(File(f, "w"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Truncate(f, 1024)
			must.NoError(t, err)
		},
		"truncate_dir_rcx": func() {
			f := filepath.Join(os.TempDir(), random())
			err := os.WriteFile(f, []byte("hello"), 0644)
			must.NoError(t, err)
			dir := filepath.Dir(f)
			l := New(Dir(dir, "rcx"))
			err = l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Truncate(f, 1024)
			must.Error(t, err)
		},
		"truncate_dir_w": func() {
			f := filepath.Join(os.TempDir(), random())
			err := os.WriteFile(f, []byte("hello"), 0o644)
			must.NoError(t, err)
			dir := filepath.Dir(f)
			l := New(Dir(dir, "w"))
			err = l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Truncate(f, 1024)
			must.NoError(t, err)
		},
	}

	// if we are child process, run the assigned test case
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent, launch child process
	forkAndRunEachCase(t, "TestLocker_truncate", cases)
}

func TestLocker_creates(t *testing.T) {
	cases := map[string]func(){
		"none": func() {
			l := New() // none
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			_, err = os.OpenFile("/tmp/a.txt", os.O_CREATE, 0o0644)
			must.Error(t, err)
		},
		"dir_rw": func() {
			f := filepath.Join(os.TempDir(), random())
			dir := filepath.Dir(f)
			l := New(Dir(dir, "rw"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			_, err = os.OpenFile(f, os.O_CREATE, 0o644)
			must.Error(t, err) // rw only, not create
		},
		"dir_rwc": func() {
			f := filepath.Join(os.TempDir(), random())
			dir := filepath.Dir(f)
			l := New(Dir(dir, "rwc"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			_, err = os.OpenFile(f, os.O_CREATE, 0o644)
			must.NoError(t, err)
		},
		"file_rwc": func() {
			f := filepath.Join(os.TempDir(), random())
			l := New(File(f, "rwc"))
			err := l.Lock(Mandatory)
			must.Error(t, err) // no such file
		},
	}

	// if we are child process, run the assigned test case
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_creates", cases)
}

func TestLocker_executes(t *testing.T) {
	cases := map[string]func(){
		"none": func() {
			l := New()
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			cmd := exec.Command("tests/fruits/hello.sh")
			_, err = cmd.CombinedOutput()
			must.Error(t, err) // no permission
			fmt.Println("err", err)
		},
		"bash_x": func() {
			l := New(
				Shared(),
				File("/usr/bin/bash", "rx"),
				File("/usr/bin/echo", "rx"),
			)
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			cmd := exec.Command("/usr/bin/bash", "-c", "/usr/bin/echo -n hi")
			b, err := cmd.CombinedOutput()
			must.NoError(t, err)
			must.Eq(t, "hi", string(b))
		},
		"deny_script": func() {
			l := New(
				Shared(),
				File("tests/fruits/hello.sh", "rw"),
				File("/usr/bin/bash", "rx"),
				File("/usr/bin/echo", "rx"),
			)
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			cmd := exec.Command("tests/fruits/hello.sh")
			_, err = cmd.CombinedOutput()
			must.Error(t, err)
		},
		"allow_script": func() {
			l := New(
				Shared(),
				Dir("/", "rx"),
				File("tests/fruits/hello.sh", "rx"),
				File("/usr/bin/bash", "rx"),
				File("/usr/bin/echo", "rx"),
			)
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			cmd := exec.Command("tests/fruits/hello.sh")
			b, err := cmd.CombinedOutput()
			must.NoError(t, err)
			must.Eq(t, "so you like fruit?", string(b))
		},
	}

	// if we are child process, run the assigned test case
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_executes", cases)
}

func TestLocker_deletes(t *testing.T) {
	cases := map[string]func(){
		"none": func() {
			l := New() // none
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Remove("tests/fruits/hello.sh")
			must.Error(t, err)
		},
		"rm_file": func() {
			f1 := filepath.Join(os.TempDir(), random())
			f2 := filepath.Join(os.TempDir(), random())
			writeFile(t, f1, "one", 0o644)
			writeFile(t, f2, "two", 0o644)
			l := New(File(f1, "rwc"), File(f2, "r"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Remove(f1)
			must.Error(t, err)
			err = os.Remove(f2)
			must.Error(t, err)
		},
		"rm_dir_rm_file": func() {
			f := filepath.Join(os.TempDir(), random())
			writeFile(t, f, "one", 0o644)
			l := New(Dir(filepath.Dir(f), "rwc"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.Remove(f)
			must.NoError(t, err)
		},
		"rm_dir_rwc": func() {
			d := os.TempDir()
			err := os.MkdirAll(filepath.Join(d, "/a/b/c"), 0o755)
			must.NoError(t, err)
			l := New(Dir(d, "rwc"))
			err = l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.RemoveAll(filepath.Join(d, "a/b"))
			must.NoError(t, err)
		},
		"rm_dir_rw": func() {
			d := os.TempDir()
			err := os.MkdirAll(filepath.Join(d, "/a/b/c"), 0o755)
			must.NoError(t, err)
			l := New(Dir(d, "rw"))
			err = l.Lock(Mandatory)
			must.NoError(t, err)
			err = os.RemoveAll(filepath.Join(d, "a/b"))
			must.Error(t, err)
		},
	}

	// if we are child process, run the assigned test case
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_deletes", cases)
}

func TestLocker_symlink(t *testing.T) {
	cases := map[string]func(){
		"read_escape": func() {
			d := os.TempDir()
			next := filepath.Join(d, random())
			old := "/etc/passwd"
			err := os.Symlink(old, next)
			must.NoError(t, err)

			l := New(Dir(d, "r"))
			err = l.Lock(Mandatory)
			must.NoError(t, err)

			_, err = os.ReadFile(next)
			must.Error(t, err)
		},
		"create_escape": func() {
			d := os.TempDir()
			next := filepath.Join(d, random())
			old := "/etc/passwd"

			l := New(Dir(d, "rwc"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)

			err = os.Symlink(old, next)
			must.NoError(t, err) // creating symlink is allowable;
			// FS_REFER is about hardlinks and mounts

			_, err = os.ReadFile(next)
			must.Error(t, err) // cannot read the symlink
		},
	}

	// if we are child process, run the assigned test case
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_symlink", cases)
}

func TestLocker_hardlink(t *testing.T) {
	cases := map[string]func(){
		"read_existing_link": func() {
			// a hardlink is as good as the real thing; if there is a pre-existing
			// link going outside the sandbox, we are still able to read it
			root := filepath.Join(os.TempDir(), randomDir())
			err := os.Mkdir(root, 0755)
			must.NoError(t, err)

			targetDir := filepath.Join(root, "target")
			sandboxDir := filepath.Join(root, "sandbox")

			err = os.Mkdir(targetDir, 0755)
			must.NoError(t, err)

			err = os.MkdirAll(sandboxDir, 0755)
			must.NoError(t, err)

			targetFile := filepath.Join(targetDir, "secrets.txt")
			err = os.WriteFile(targetFile, []byte("p4ssw0rd"), 0644)
			must.NoError(t, err)

			link := filepath.Join(sandboxDir, "link.txt")

			err = os.Link(targetFile, link)
			must.NoError(t, err)

			l := New(Dir(sandboxDir, "r"))
			err = l.Lock(Mandatory)
			must.NoError(t, err)

			_, err = os.ReadFile(link)
			must.NoError(t, err)
		},
		"create_escape_link": func() {
			d := os.TempDir()
			next := filepath.Join(d, random())

			l := New(Dir(d, "rwc"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)

			target := filepath.Join(os.Getenv("HOME"), ".profile")

			// creating a hardlink outside the sandbox is blocked
			err = os.Link(target, next)
			must.Error(t, err)
		},
		"create_internal_link": func() {
			d := os.TempDir()

			next := filepath.Join(d, random())
			old := filepath.Join(d, random())
			writeFile(t, old, "hello", 0o644)

			l := New(Dir(d, "rwc"))
			err := l.Lock(Mandatory)
			must.NoError(t, err)

			// creating a hardlink within the sandbox is fine
			err = os.Link(old, next)
			must.NoError(t, err)

			// reading the hardlink should be fine
			_, err = os.ReadFile(next)
			must.NoError(t, err)
		},
	}

	// if we are child process we run the assigned case and then exit
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_hardlink", cases)
}

func TestLocker_mount(t *testing.T) {
	if syscall.Geteuid() != 0 {
		t.Skip("must be root to run mount tests")
	}

	cases := map[string]func(){
		"read_existing_mount": func() {
			d := os.TempDir()
			mountpoint := filepath.Join(d, random())
			_, err := os.Create(mountpoint)
			must.NoError(t, err)

			// setup the mountpoint with a file that should be overridden
			writeFile(t, "should be overridden", mountpoint, 0o644)

			cmd := exec.Command("mount", "--bind", "/etc/os-release", "--target", mountpoint)
			output, cmdErr := cmd.CombinedOutput()
			fmt.Println("output", string(output))
			must.NoError(t, cmdErr, must.Sprintf("mount failure: %s", string(output)))

			// sandbox excludes mount source
			l := New(Dir(d, "rwc"))
			lockErr := l.Lock(Mandatory)
			must.NoError(t, lockErr)

			// we can read the mount; exists before the lockdown
			// and prove the mount source is actually mounted over the original
			b, readErr := os.ReadFile(mountpoint)
			must.NoError(t, readErr)
			must.StrContains(t, string(b), "PRETTY_NAME")
			must.StrNotContains(t, string(b), "should be overridden")
		},
		"create_escaping_mount": func() {
			d := os.TempDir()
			mountpoint := filepath.Join(d, random())
			_, err := os.Create(mountpoint)
			must.NoError(t, err)

			source := "/etc/os-release"

			// sandbox excludes mount source
			l := New(Dir(d, "rwc"))
			lockErr := l.Lock(Mandatory)
			must.NoError(t, lockErr)

			// landlock will prevent creating this mount
			cmd := exec.Command("mount", "--bind", source, "--target", mountpoint)
			_, cmdErr := cmd.CombinedOutput()
			must.ErrorContains(t, cmdErr, "permission denied")
		},
	}

	// if we are child process we run the assigned case and then exit
	if isChildRunner(cases) {
		return
	}

	// otherwise if we are parent process, launch child processes
	forkAndRunEachCase(t, "TestLocker_mount", cases)
}

func writeFile(t *testing.T, path, content string, mode fs.FileMode) {
	err := os.WriteFile(path, []byte(content), mode)
	must.NoError(t, err)
}

func tmpFile(t *testing.T, name, content string) string {
	dir := os.TempDir() // cannot use t.TempDir
	f := filepath.Join(dir, name)
	err := os.WriteFile(f, []byte(content), 0o644)
	must.NoError(t, err)
	return f
}

// random will create a random text file name
func random() string {
	n := 6
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte(rand.Int()%26 + 97)
	}
	return string(b) + ".txt"
}

func randomDir() string {
	n := 6
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte(rand.Int()%26 + 97)
	}
	return string(b)
}

// This part gets run in each sub-process; it is the actual test
// case, and must return non-zero on test failure.
func isChildRunner(cases map[string]func()) bool {
	if name := os.Getenv("TEST"); name != "" {
		f := cases[name]
		f()
		return true
	}
	return false
}

// This part is the normal test runner. It launches a sub-process
// for each test case so we can observe landlock behavior more than
// just once.
func forkAndRunEachCase(t *testing.T, prefix string, cases map[string]func()) {
	for name := range cases {
		arg := fmt.Sprintf("-test.run=%s/%s", prefix, name)
		cmd := exec.Command(os.Args[0], arg)
		cmd.Env = append(os.Environ(), fmt.Sprintf("TEST=%s", name))
		b, err := cmd.CombinedOutput()
		t.Logf("TEST[%s] (arg: %s)\n\t|> %s\n\n", name, arg, string(b))
		must.NoError(t, err)
	}
}

func requiresVersion(t *testing.T, v int) {
	if version < v {
		t.Skipf("version of landlock not high enough for test; need %d, got %d", v, version)
	}
}
