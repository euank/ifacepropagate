package case01

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"ifacepropagate.testcase/test01/pkg"
)

type frobulatingReader struct {
	bytes.Buffer
}

func (f *frobulatingReader) Frobulate() {}

var _ io.Reader = &frobulatingReader{}
var _ pkg.Frobulator = &frobulatingReader{}

func TestCase01(t *testing.T) {
	normalReader := &bytes.Buffer{}
	frobingReader := &frobulatingReader{}

	wrappedNormal := readFrobulator{normalReader}.propagateInterfaces()
	wrappedFrobber := readFrobulator{frobingReader}.propagateInterfaces()

	_, frobs := wrappedNormal.(pkg.Frobulator)
	require.False(t, frobs)
	_, frobs = wrappedFrobber.(pkg.Frobulator)
	require.True(t, frobs)

}
