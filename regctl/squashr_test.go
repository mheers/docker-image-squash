package regctl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSquash(t *testing.T) {
	err := Squash("alpine", "alpine.tar")
	require.NoError(t, err)
}
