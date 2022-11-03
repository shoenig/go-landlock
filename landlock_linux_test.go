//go:build linux

package landlock

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
			failure: []string{"tests/Labels.txt", "tests/fruits", "tests/fruits/apple.txt"},
		},
		{
			name:    "read top file",
			paths:   []string{"f:r:tests/Labels.txt"},
			success: []string{"tests/Labels.txt"},
			failure: []string{"tests/fruits/apple.txt"},
		},
		{
			name:    "read sub file",
			paths:   []string{"d:r:tests"},
			success: []string{"tests/fruits/apple.txt"},
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
		err := New(paths...).Lock(Enforce)
		must.NoError(t, err)

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
		continue

	}
}
