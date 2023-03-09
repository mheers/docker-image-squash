package squash

import (
	"encoding/json"
	"os"
	"path"

	"github.com/mheers/docker-image-squash/helpers"
)

func Squash(inputTar, outputTar string) error {
	tmpDir, err := os.MkdirTemp("", "squash")
	if err != nil {
		return err
	}

	origDir := path.Join(tmpDir, "orig")
	if err := os.Mkdir(origDir, 0755); err != nil {
		return err
	}

	if err := helpers.Untar(inputTar, origDir); err != nil {
		return err
	}

	layers, err := GetLayersOfExtractedImage(origDir)
	if err != nil {
		return err
	}

	dstDir := path.Join(tmpDir, "dst")
	for _, layer := range layers {
		if err := helpers.Untar(path.Join(origDir, layer), dstDir); err != nil {
			return err
		}
	}

	output := path.Join(tmpDir, "output.tar")
	if err := helpers.Tar(dstDir, output); err != nil {
		return err
	}

	if err := os.Rename(output, outputTar); err != nil {
		return err
	}

	return nil
}

func GetLayers(data []byte) ([]string, error) {
	type Manifest struct {
		Layers []string `json:"Layers"`
	}
	var manifest []Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest[0].Layers, nil
}

func GetLayersOfExtractedImage(dir string) ([]string, error) {
	data, err := os.ReadFile(path.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	return GetLayers(data)
}
