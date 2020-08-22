package ifacepropagate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIfacePropgate(t *testing.T) {
	testdirs, err := filepath.Glob("./testdata/*")
	require.Nil(t, err)

	for _, dir := range testdirs {
	}
}
