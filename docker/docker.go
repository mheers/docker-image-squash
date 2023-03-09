package docker

import "github.com/mheers/docker-image-squash/helpers"

func Export(image, outputTar string) error {
	// TODO: check if image exists
	// TODO: support for other runtimes
	// TODO: implement directly in Go
	return helpers.Run("docker", "save", "-o", outputTar, image)
}

func Pull(image string) error {
	// TODO: support for other runtimes
	// TODO: implement directly in Go
	return helpers.Run("docker", "pull", image)
}
