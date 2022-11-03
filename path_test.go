package landlock

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestPath_Dir(t *testing.T) {
	cases := []struct {
		mode string
		path string
		exp  *Path
	}{
		{
			mode: "r",
			path: "/etc",
			exp:  &Path{mode: "r", path: "/etc", dir: true},
		},
		{
			mode: "rx",
			path: "/opt/bin",
			exp:  &Path{mode: "rx", path: "/opt/bin", dir: true},
		},
	}

	for _, tc := range cases {
		result := Dir(tc.path, tc.mode)
		must.Equal(t, tc.exp, result)
	}
}

func TestPath_ParsePath(t *testing.T) {
	cases := []struct {
		input string
		exp   *Path
	}{
		{
			input: "d:r:/etc",
			exp:   &Path{mode: "r", path: "/etc", dir: true},
		},
		{
			input: "d:rw:/etc/system",
			exp:   &Path{mode: "rw", path: "/etc/system", dir: true},
		},
	}

	for _, tc := range cases {
		result, err := ParsePath(tc.input)
		must.NoError(t, err)
		must.Equal(t, tc.exp, result)
	}
}

func TestPath_ParsePath_error(t *testing.T) {
	cases := []struct {
		input string
		exp   error
	}{
		{
			input: "",
			exp:   ErrImproperPath,
		},
		{
			input: "rw:",
			exp:   ErrImproperPath,
		},
		{
			input: "z:/etc",
			exp:   ErrImproperPath,
		},
		{
			input: "rw:./foo/..",
			exp:   ErrImproperPath,
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ParsePath(tc.input)
			must.Nil(t, result)
			must.ErrorIs(t, err, tc.exp,
				must.Sprintf("err: %#v", err),
				must.Sprintf("exp: %#v", tc.exp),
			)
		})
	}
}

func TestPath_IsProperMode(t *testing.T) {
	cases := []struct {
		input string
		exp   bool
	}{
		{input: "r", exp: true},
		{input: "w", exp: true},
		{input: "c", exp: true},
		{input: "x", exp: true},
		{input: "rw", exp: true},
		{input: "wrc", exp: true},
		{input: "xc", exp: true},
		{input: "", exp: false},
		{input: "a", exp: false},
		{input: "rwa", exp: false},
		{input: "xar", exp: false},
		{input: "RW", exp: false},
		{input: "r w c x", exp: false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := IsProperMode(tc.input)
			must.EqOp(t, tc.exp, result)
		})
	}
}

func TestPath_IsProperPath(t *testing.T) {
	cases := []struct {
		input string
		exp   bool
	}{
		{input: "/", exp: true},
		{input: "", exp: false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := IsProperPath(tc.input)
			must.EqOp(t, tc.exp, result)
		})
	}
}
