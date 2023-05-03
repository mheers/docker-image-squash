package regctl

import (
	"context"
	"fmt"
	"strings"

	"github.com/mheers/docker-image-squash/helpers"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
)

func Squash(image, outputDir string) error {
	ctx := context.Background()
	r, err := ref.New(image)
	if err != nil {
		return err
	}

	rc := newRegClient()
	defer rc.Close(ctx, r)

	// make it recursive for index of index scenarios
	m, err := rc.ManifestGet(ctx, r)
	if err != nil {
		return err
	}
	if m.IsList() {
		if imageOpts.platform == "" {
			imageOpts.platform = "local"
		}
		plat, err := platform.Parse(imageOpts.platform)
		if err != nil {
			log.WithFields(logrus.Fields{
				"platform": imageOpts.platform,
				"err":      err,
			}).Warn("Could not parse platform")
		}
		desc, err := manifest.GetPlatformDesc(m, &plat)
		if err != nil {
			pl, _ := manifest.GetPlatformList(m)
			var ps []string
			for _, p := range pl {
				ps = append(ps, p.String())
			}
			log.WithFields(logrus.Fields{
				"platform":  plat,
				"err":       err,
				"platforms": strings.Join(ps, ", "),
			}).Warn("Platform could not be found in manifest list")
			return err
		}
		m, err = rc.ManifestGet(ctx, r, regclient.WithManifestDesc(*desc))
		if err != nil {
			return fmt.Errorf("failed to pull platform specific digest: %w", err)
		}
	}
	// go through layers in reverse
	mi, ok := m.(manifest.Imager)
	if !ok {
		return fmt.Errorf("reference is not a known image media type")
	}
	layers, err := mi.GetLayers()
	if err != nil {
		return err
	}

	for i, layer := range layers {
		blob, err := rc.BlobGet(ctx, r, layer)
		if err != nil {
			return fmt.Errorf("failed pulling layer %d: %w", i, err)
		}
		btr, err := blob.ToTarReader()
		if err != nil {
			return fmt.Errorf("could not convert layer %d to tar reader: %w", i, err)
		}
		tr, err := btr.GetTarReader()
		if err != nil {
			return fmt.Errorf("could not get tar reader for layer %d: %w", i, err)
		}

		if err := helpers.UntarTarReader(tr, outputDir); err != nil {
			return fmt.Errorf("failed squashing layer %d: %w", i, err)
		}

	}

	return nil
}
