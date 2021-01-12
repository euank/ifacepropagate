package case02

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type recorder struct {
	calls []string
}

func (r *recorder) Method1() {
	r.calls = append(r.calls, "1")
}
func (r *recorder) Method2() {
	r.calls = append(r.calls, "2")
}
func (r *recorder) Method3() {
	r.calls = append(r.calls, "3")
}
func (r *recorder) Method4() {
	r.calls = append(r.calls, "4")
}

func TestPartlyOverridden(t *testing.T) {
	r := &recorder{}
	s := new(r)

	_, impl2 := s.(If2)
	require.True(t, impl2)

	s.Method1()
	s.Method2()
	s.(If2).Method3()
	s.(If2).Method4()
	// partialOverride should have gotten method 1 and 3, so we shouldn't record those
	require.EqualValues(t, []string{"2", "4"}, r.calls)
}
