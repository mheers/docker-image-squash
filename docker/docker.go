package docker

import (
	"github.com/mheers/docker-image-squash/regctl"
)

func Export(image, outputTar string) error {
	if err := regctl.ExportImage(image, outputTar); err != nil {
		return err
	}
	return nil
}
