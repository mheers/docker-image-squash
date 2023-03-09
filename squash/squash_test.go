package squash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSquash(t *testing.T) {
	err := Squash("../cmd/test.tar", "test-squashed.tar")
	require.NoError(t, err)
}

func TestGetLayers(t *testing.T) {
	layers, err := GetLayers([]byte(`[{"Config":"77414ae281db98275afa7659731a495fb933ede750b9d758e8dd17891b976ad6.json","RepoTags":["test:latest"],"Layers":["df13cc887c6ff0faabec81bc4bb8b72900db35e7d107e66a56b500eb0845396e/layer.tar","a0cb8277c94ef0ce6a52219b419c19211227757f85d1a80cc64c256aa1a502a2/layer.tar","c6d2d3dbb97a114b523674aaf9c35cfa04dcf4ba2936bc350050e015a0391073/layer.tar","dc509d83dbc24e15bab860dcb0e4c2f18c15369beabea0e5661b6935de2ee5df/layer.tar"]}]`))
	require.NoError(t, err)
	require.Len(t, layers, 4)
}
