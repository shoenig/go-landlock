//go:build !linux

package landlock

func Test_Available(t *testing.T) {
	a := Available()
	must.False(t, a)
}
