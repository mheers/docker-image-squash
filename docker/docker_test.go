package docker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("alpine.tar")
	})
	require.NoFileExists(t, "alpine.tar")
	err := Export("alpine", "alpine.tar")

	require.NoError(t, err)
	require.FileExists(t, "alpine.tar")
}
