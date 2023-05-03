package regctl

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/mod"
	"github.com/regclient/regclient/pkg/template"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var imageOpts struct {
	checkBaseRef    string
	checkBaseDigest string
	checkSkipConfig bool
	create          string
	exportRef       string
	forceRecursive  bool
	format          string
	formatFile      string
	includeExternal bool
	digestTags      bool
	list            bool
	modOpts         []mod.Opts
	platform        string
	platforms       []string
	referrers       bool
	replace         bool
	requireList     bool
}

func ExportImage(image, outputTar string) error {
	rc := newRegClient()
	r, err := ref.New(image)
	if err != nil {
		return err
	}
	defer rc.Close(context.Background(), r)

	f, err := os.Create(outputTar)
	if err != nil {
		return err
	}
	return rc.ImageExport(context.Background(), r, f)
}

func runImageExport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	var w io.Writer
	if len(args) == 2 {
		w, err = os.Create(args[1])
		if err != nil {
			return err
		}
	} else {
		w = cmd.OutOrStdout()
	}
	rc := newRegClient()
	defer rc.Close(ctx, r)
	opts := []regclient.ImageOpts{}
	if imageOpts.platform != "" {
		p, err := platform.Parse(imageOpts.platform)
		if err != nil {
			return err
		}
		m, err := rc.ManifestGet(ctx, r)
		if err != nil {
			return err
		}
		if m.IsList() {
			d, err := manifest.GetPlatformDesc(m, &p)
			if err != nil {
				return err
			}
			r.Digest = d.Digest.String()
		}
	}
	if imageOpts.exportRef != "" {
		eRef, err := ref.New(imageOpts.exportRef)
		if err != nil {
			return fmt.Errorf("cannot parse %s: %w", imageOpts.exportRef, err)
		}
		opts = append(opts, regclient.ImageWithExportRef(eRef))
	}
	log.WithFields(logrus.Fields{
		"ref": r.CommonName(),
	}).Debug("Image export")
	return rc.ImageExport(ctx, r, w, opts...)
}

func runImageGetFile(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	filename := args[1]
	filename = strings.TrimPrefix(filename, "/")
	rc := newRegClient()
	defer rc.Close(ctx, r)

	log.WithFields(logrus.Fields{
		"ref":      r.CommonName(),
		"filename": filename,
	}).Debug("Get file")

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
	for i := len(layers) - 1; i >= 0; i-- {
		blob, err := rc.BlobGet(ctx, r, layers[i])
		if err != nil {
			return fmt.Errorf("failed pulling layer %d: %w", i, err)
		}
		btr, err := blob.ToTarReader()
		if err != nil {
			return fmt.Errorf("could not convert layer %d to tar reader: %w", i, err)
		}
		th, rdr, err := btr.ReadFile(filename)
		if err != nil {
			if errors.Is(err, types.ErrFileNotFound) {
				continue
			}
			return fmt.Errorf("failed pulling from layer %d: %w", i, err)
		}
		// file found, output
		if imageOpts.formatFile != "" {
			data := struct {
				Header *tar.Header
				Reader io.Reader
			}{
				Header: th,
				Reader: rdr,
			}
			return template.Writer(cmd.OutOrStdout(), imageOpts.formatFile, data)
		}
		var w io.Writer
		if len(args) < 3 {
			w = cmd.OutOrStdout()
		} else {
			w, err = os.Create(args[2])
			if err != nil {
				return err
			}
		}
		_, err = io.Copy(w, rdr)
		if err != nil {
			return err
		}
		return nil
	}
	// all layers exhausted, not found or deleted
	return types.ErrNotFound
}

func runImageImport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	rs, err := os.Open(args[1])
	if err != nil {
		return err
	}
	defer rs.Close()
	rc := newRegClient()
	defer rc.Close(ctx, r)
	log.WithFields(logrus.Fields{
		"ref":  r.CommonName(),
		"file": args[1],
	}).Debug("Image import")

	return rc.ImageImport(ctx, r, rs)
}

func runImageInspect(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	rc := newRegClient()
	defer rc.Close(ctx, r)

	log.WithFields(logrus.Fields{
		"host":     r.Registry,
		"repo":     r.Repository,
		"tag":      r.Tag,
		"platform": imageOpts.platform,
	}).Debug("Image inspect")

	manifestOpts.platform = imageOpts.platform
	if !flagChanged(cmd, "list") {
		manifestOpts.list = false
	}

	m, err := getManifest(ctx, rc, r)
	if err != nil {
		return err
	}
	mi, ok := m.(manifest.Imager)
	if !ok {
		return fmt.Errorf("manifest does not support image methods%.0w", types.ErrUnsupportedMediaType)
	}
	cd, err := mi.GetConfig()
	if err != nil {
		return err
	}

	blobConfig, err := rc.BlobGetOCIConfig(ctx, r, cd)
	if err != nil {
		return err
	}
	switch imageOpts.format {
	case "raw":
		imageOpts.format = "{{ range $key,$vals := .RawHeaders}}{{range $val := $vals}}{{printf \"%s: %s\\n\" $key $val }}{{end}}{{end}}{{printf \"\\n%s\" .RawBody}}"
	case "rawBody", "raw-body", "body":
		imageOpts.format = "{{printf \"%s\" .RawBody}}"
	case "rawHeaders", "raw-headers", "headers":
		imageOpts.format = "{{ range $key,$vals := .RawHeaders}}{{range $val := $vals}}{{printf \"%s: %s\\n\" $key $val }}{{end}}{{end}}"
	}
	return template.Writer(cmd.OutOrStdout(), imageOpts.format, blobConfig)
}

func runImageMod(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	var rNew ref.Ref
	if imageOpts.create != "" {
		if strings.ContainsAny(imageOpts.create, "/:") {
			rNew, err = ref.New((imageOpts.create))
			if err != nil {
				return fmt.Errorf("failed to parse new image name %s: %w", imageOpts.create, err)
			}
		} else {
			rNew = r
			rNew.Digest = ""
			rNew.Tag = imageOpts.create
		}
	} else if imageOpts.replace {
		if r.Tag == "" {
			return fmt.Errorf("cannot replace an image digest, must include a tag")
		}
		rNew = r
		rNew.Digest = ""
	}
	rc := newRegClient()

	log.WithFields(logrus.Fields{
		"ref": r.CommonName(),
	}).Debug("Modifying image")

	defer rc.Close(ctx, r)
	rOut, err := mod.Apply(ctx, rc, r, imageOpts.modOpts...)
	if err != nil {
		return err
	}
	if rNew.Tag != "" {
		defer rc.Close(ctx, rNew)
		err = rc.ImageCopy(ctx, rOut, rNew)
		if err != nil {
			return fmt.Errorf("failed copying image to new name: %w", err)
		}
		fmt.Printf("%s\n", rNew.CommonName())
	} else {
		fmt.Printf("%s\n", rOut.CommonName())
	}
	return nil
}

func runImageRateLimit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	rc := newRegClient()

	log.WithFields(logrus.Fields{
		"host": r.Registry,
		"repo": r.Repository,
		"tag":  r.Tag,
	}).Debug("Image rate limit")

	// request only the headers, avoids adding to Docker Hub rate limits
	m, err := rc.ManifestHead(ctx, r)
	if err != nil {
		return err
	}

	return template.Writer(cmd.OutOrStdout(), imageOpts.format, manifest.GetRateLimit(m))
}

type modFlagFunc struct {
	f func(string) error
	t string
}

func (m *modFlagFunc) IsBoolFlag() bool {
	return m.t == "bool"
}

func (m *modFlagFunc) String() string {
	return ""
}

func (m *modFlagFunc) Set(val string) error {
	return m.f(val)
}

func (m *modFlagFunc) Type() string {
	return m.t
}
