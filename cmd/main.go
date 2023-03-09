package main

import (
	"os"
	"path"

	"github.com/mheers/docker-image-squash/docker"
	"github.com/mheers/docker-image-squash/squash"
)

func main() {
	if len(os.Args) != 3 {
		panic("usage: docker-image-squash <image> <output.tar>")
	}

	image := os.Args[1]
	output := os.Args[2]

	tmpDir, err := os.MkdirTemp("", "squash")
	if err != nil {
		panic(err)
	}

	tmpExportFile := path.Join(tmpDir, "export.tar")

	err = docker.Pull(image)
	if err != nil {
		panic(err)
	}

	err = docker.Export(image, tmpExportFile)
	if err != nil {
		panic(err)
	}

	err = squash.Squash(tmpExportFile, output)
	if err != nil {
		panic(err)
	}
}
