package regctl

import (
	"context"
	"fmt"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/pkg/template"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest <cmd>",
	Short: "manage manifests",
}

var manifestGetCmd = &cobra.Command{
	Use:               "get <image_ref>",
	Aliases:           []string{"pull"},
	Short:             "retrieve manifest or manifest list",
	Long:              `Shows the manifest or manifest list of the specified image.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeArgTag,
	RunE:              runManifestGet,
}

var manifestHeadCmd = &cobra.Command{
	Use:               "head <image_ref>",
	Aliases:           []string{"digest"},
	Short:             "http head request for manifest",
	Long:              `Shows the digest or headers from an http manifest head request.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeArgTag,
	RunE:              runManifestHead,
}

var manifestOpts struct {
	byDigest      bool
	contentType   string
	diffCtx       int
	diffFullCtx   bool
	forceTagDeref bool
	formatGet     string
	formatHead    string
	formatPut     string
	list          bool
	platform      string
	referrers     bool
	requireDigest bool
	requireList   bool
}

func getManifest(ctx context.Context, rc *regclient.RegClient, r ref.Ref) (manifest.Manifest, error) {
	m, err := rc.ManifestGet(context.Background(), r)
	if err != nil {
		return m, err
	}

	// add warning if not list and list required or platform requested
	if !m.IsList() && manifestOpts.requireList {
		log.Warn("Manifest list unavailable")
		return m, ErrNotFound
	}
	if !m.IsList() && manifestOpts.platform != "" {
		log.Info("Manifest list unavailable, ignoring platform flag")
	}

	// retrieve the specified platform from the manifest list
	if m.IsList() && !manifestOpts.list && !manifestOpts.requireList {
		desc, err := getPlatformDesc(ctx, rc, m)
		if err != nil {
			return m, fmt.Errorf("failed to lookup platform specific digest: %w", err)
		}
		m, err = rc.ManifestGet(ctx, r, regclient.WithManifestDesc(*desc))
		if err != nil {
			return m, fmt.Errorf("failed to pull platform specific digest: %w", err)
		}
	}
	return m, nil
}

func getPlatformDesc(ctx context.Context, rc *regclient.RegClient, m manifest.Manifest) (*types.Descriptor, error) {
	var desc *types.Descriptor
	var err error
	if !m.IsList() {
		return desc, fmt.Errorf("%w: manifest is not a list", ErrInvalidInput)
	}
	if !m.IsSet() {
		m, err = rc.ManifestGet(ctx, m.GetRef())
		if err != nil {
			return desc, fmt.Errorf("unable to retrieve manifest list: %w", err)
		}
	}

	var plat platform.Platform
	if manifestOpts.platform != "" && manifestOpts.platform != "local" {
		plat, err = platform.Parse(manifestOpts.platform)
		if err != nil {
			log.WithFields(logrus.Fields{
				"platform": manifestOpts.platform,
				"err":      err,
			}).Warn("Could not parse platform")
		}
	}
	if plat.OS == "" {
		plat = platform.Local()
	}
	desc, err = manifest.GetPlatformDesc(m, &plat)
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
		return desc, ErrNotFound
	}
	log.WithFields(logrus.Fields{
		"platform": plat,
		"digest":   desc.Digest.String(),
	}).Debug("Found platform specific digest in manifest list")
	return desc, nil
}

func runManifestHead(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if manifestOpts.platform != "" && !flagChanged(cmd, "list") {
		manifestOpts.list = false
	} else if !manifestOpts.list && !flagChanged(cmd, "list") {
		manifestOpts.list = true
	}

	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	rc := newRegClient()

	log.WithFields(logrus.Fields{
		"host": r.Registry,
		"repo": r.Repository,
		"tag":  r.Tag,
	}).Debug("Manifest head")

	mOpts := []regclient.ManifestOpts{}
	if manifestOpts.requireDigest || (!flagChanged(cmd, "require-digest") && !flagChanged(cmd, "format")) {
		mOpts = append(mOpts, regclient.WithManifestRequireDigest())
	}

	// attempt to request only the headers, avoids Docker Hub rate limits
	m, err := rc.ManifestHead(ctx, r, mOpts...)
	if err != nil {
		return err
	}

	// add warning if not list and list required or platform requested
	if !m.IsList() && manifestOpts.requireList {
		log.Warn("Manifest list unavailable")
		return ErrNotFound
	}
	if !m.IsList() && manifestOpts.platform != "" {
		log.Info("Manifest list unavailable, ignoring platform flag")
	}

	// retrieve the specified platform from the manifest list
	for m.IsList() && !manifestOpts.list && !manifestOpts.requireList {
		desc, err := getPlatformDesc(ctx, rc, m)
		if err != nil {
			return fmt.Errorf("failed retrieving platform specific digest: %w", err)
		}
		r.Digest = desc.Digest.String()
		m, err = rc.ManifestHead(ctx, r, mOpts...)
		if err != nil {
			return fmt.Errorf("failed retrieving platform specific digest: %w", err)
		}
	}

	switch manifestOpts.formatHead {
	case "", "digest":
		manifestOpts.formatHead = "{{ printf \"%s\\n\" .GetDescriptor.Digest }}"
	case "rawHeaders", "raw-headers", "headers":
		manifestOpts.formatHead = "{{ range $key,$vals := .RawHeaders}}{{range $val := $vals}}{{printf \"%s: %s\\n\" $key $val }}{{end}}{{end}}"
	}
	return template.Writer(cmd.OutOrStdout(), manifestOpts.formatHead, m)
}

func runManifestGet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if manifestOpts.platform != "" && !flagChanged(cmd, "list") {
		manifestOpts.list = false
	} else if !manifestOpts.list && !flagChanged(cmd, "list") {
		manifestOpts.list = true
	}

	r, err := ref.New(args[0])
	if err != nil {
		return err
	}
	rc := newRegClient()
	defer rc.Close(ctx, r)

	m, err := getManifest(ctx, rc, r)
	if err != nil {
		return err
	}

	switch manifestOpts.formatGet {
	case "raw":
		manifestOpts.formatGet = "{{ range $key,$vals := .RawHeaders}}{{range $val := $vals}}{{printf \"%s: %s\\n\" $key $val }}{{end}}{{end}}{{printf \"\\n%s\" .RawBody}}"
	case "rawBody", "raw-body", "body":
		manifestOpts.formatGet = "{{printf \"%s\" .RawBody}}"
	case "rawHeaders", "raw-headers", "headers":
		manifestOpts.formatGet = "{{ range $key,$vals := .RawHeaders}}{{range $val := $vals}}{{printf \"%s: %s\\n\" $key $val }}{{end}}{{end}}"
	}
	return template.Writer(cmd.OutOrStdout(), manifestOpts.formatGet, m)
}
