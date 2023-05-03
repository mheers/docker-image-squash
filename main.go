package main

import (
	"os"

	"github.com/mheers/docker-image-squash/helpers"
	"github.com/mheers/docker-image-squash/regctl"
)

func main() {
	if len(os.Args) != 3 {
		panic("usage: docker-image-squash <image> <output.tar>")
	}

	image := os.Args[1]
	output := os.Args[2]

	// create a temporary directory to store the layers
	tmpDir, err := os.MkdirTemp("", "regctl-squashr")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// squash the image
	if err := regctl.Squash(image, tmpDir); err != nil {
		panic(err)
	}

	// create the output tarball
	if err := helpers.Tar(tmpDir, output); err != nil {
		panic(err)
	}
}
