package regctl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSquash(t *testing.T) {
	err := Squash("mheers/test", "test.tar")
	require.NoError(t, err)
}
