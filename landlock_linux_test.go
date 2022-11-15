//go:build linux

package landlock

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/shoenig/test/must"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

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
	cases := []struct {
		name    string
		paths   []string
		success []string
		failure []string
	}{
		{
			name:    "none",
			paths:   nil,
			success: nil,
			failure: []string{
				"tests/Labels.txt",
				"tests/fruits",
				"tests/fruits/apple.txt",
			},
		},
		{
			name:  "write only file",
			paths: []string{"f:w:tests/Labels.txt"},
			failure: []string{
				"tests/Labels.txt",
				"tests/fruits/apple.txt",
			},
		},
		{
			name:  "write only dir",
			paths: []string{"d:w:tests/"},
			failure: []string{
				"tests/Labels.txt",
				"tests/fruits/apple.txt",
			},
		},
		{
			name:    "read top file",
			paths:   []string{"f:r:tests/Labels.txt"},
			success: []string{"tests/Labels.txt"},
			failure: []string{
				"tests/fruits/apple.txt",
				"tests/fruits/banana.txt",
				"tests/veggies/celary.txt",
				"tests/veggies/corn.txt",
				"tests/veggies/unsure/beans.txt",
			},
		},
		{
			name:  "read top dir",
			paths: []string{"d:r:tests"},
			success: []string{
				"tests/Labels.txt",
				"tests/fruits/apple.txt",
				"tests/fruits/banana.txt",
				"tests/veggies/celary.txt",
				"tests/veggies/corn.txt",
				"tests/veggies/unsure/beans.txt",
			},
		},
		{
			name:  "read fruits dir",
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
		},
		{
			name:  "read beans file",
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
		},
		{
			name: "mixed file",
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
		},
		{
			name:    "etc directory",
			paths:   []string{"d:r:/etc"},
			success: []string{"/etc/passwd"},
			failure: []string{"/bin/sh"},
		},
	}

	try := func(num int) {
		tc := cases[num]
		fmt.Println("running test case:", num, tc.name)
		var paths []*Path
		for _, path := range tc.paths {
			p, err := ParsePath(path)
			must.NoError(t, err)
			paths = append(paths, p)
		}
		err := New(paths...).Lock(Mandatory)
		must.NoError(t, err, must.Sprint("paths", paths))

		for _, p := range tc.failure {
			_, err := os.ReadFile(p)
			must.Error(t, err)
			must.StrHasSuffix(t, "permission denied", err.Error())
		}
		for _, p := range tc.success {
			_, err := os.ReadFile(p)
			must.NoError(t, err)
		}
	}

	// This part gets run in each sub-process. It is the actual
	// test case, and must return non-zero on test case failure.
	// Using t to fail is fine.
	if env := os.Getenv("TEST"); env != "" {
		num, _ := strconv.Atoi(env)
		try(num)
		return
	}

	// This part is the normal test runner. It launches a sub-process
	// for each test case, so we can observe .Lock() behavior more
	// than just once.
	for i, tc := range cases {
		arg := fmt.Sprintf("-test.run=TestLocker_reads/%s", tc.name)
		cmd := exec.Command(os.Args[0], arg)
		cmd.Env = append(os.Environ(), fmt.Sprintf("TEST=%d", i))
		b, err := cmd.CombinedOutput()
		t.Logf("TEST[%d] (arg: %s)\n\t|> %s\n\n", i, arg, string(b))
		must.NoError(t, err)
	}
}

func tmpFile(t *testing.T, name, content string) string {
	dir := os.TempDir() // cannot use t.TempDir
	f := filepath.Join(dir, name)
	err := os.WriteFile(f, []byte(content), 0o644)
	must.NoError(t, err)
	return f
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

	// This part gets run in each sub-process; it is the actual test
	// case, and must return non-zero on test failure.
	if name := os.Getenv("TEST"); name != "" {
		f := cases[name]
		f()
		return
	}

	// This part is the normal test runner. It launches a sub-process
	// for each test case so we can observe landlock behavior more than
	// just once.
	for name := range cases {
		arg := fmt.Sprintf("-test.run=TestLocker_writes/%s", name)
		cmd := exec.Command(os.Args[0], arg)
		cmd.Env = append(os.Environ(), fmt.Sprintf("TEST=%s", name))
		b, err := cmd.CombinedOutput()
		t.Logf("TEST[%s] (arg: %s)\n\t|> %s\n\n", name, arg, string(b))
		must.NoError(t, err)
	}
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

	if name := os.Getenv("TEST"); name != "" {
		f := cases[name]
		f()
		return
	}

	for name := range cases {
		arg := fmt.Sprintf("-test.run=TestLocker_creates/%s", name)
		cmd := exec.Command(os.Args[0], arg)
		cmd.Env = append(os.Environ(), fmt.Sprintf("TEST=%s", name))
		b, err := cmd.CombinedOutput()
		t.Logf("TEST[%s] (arg: %s)\n\t|> %s\n\n", name, arg, string(b))
		must.NoError(t, err)
	}
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

	if name := os.Getenv("TEST"); name != "" {
		f := cases[name]
		f()
		return
	}

	for name := range cases {
		arg := fmt.Sprintf("-test.run=TestLocker_executes/%s", name)
		cmd := exec.Command(os.Args[0], arg)
		cmd.Env = append(os.Environ(), fmt.Sprintf("TEST=%s", name))
		b, err := cmd.CombinedOutput()
		t.Logf("TEST[%s] (arg: %s)\n\t|> %s\n\n", name, arg, string(b))
		must.NoError(t, err)
	}
}
