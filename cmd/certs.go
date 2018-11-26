package cmd

import (
	"os"
	"path/filepath"

	"github.com/clearlinux/diva/diva"
	"github.com/clearlinux/diva/internal/config"
	"github.com/clearlinux/diva/internal/helpers"
	"github.com/clearlinux/diva/pkginfo"
	"github.com/spf13/cobra"
)

type sigCmdFlags struct {
	mixName  string
	version  string
	latest   bool
	certpath string
}

var sigFlags sigCmdFlags

func init() {
	sigCmd.Flags().StringVarP(&sigFlags.mixName, "name", "n", "clear", "name of data group")
	sigCmd.Flags().StringVarP(&sigFlags.version, "version", "v", "0", "version to check")
	sigCmd.Flags().BoolVar(&sigFlags.latest, "latest", false, "get the latest version from upstreamURL")
	sigCmd.Flags().StringVar(&sigFlags.certpath, "certpath", "/usr/share/clear/update-ca/Swupd_Root.pem", "fully qualified path to ca-cert")
}

var sigCmd = &cobra.Command{
	Use:   "signature",
	Short: "validates Manifest.MoM with the Manifest.MoM.sig",
	Long: `validates all of the manifests and their files by validating the
Manifest.MoM.sig file with the ca-cert`,
	Run: runCheckCerts,
}

func runCheckCerts(cmd *cobra.Command, args []string) {
	r := diva.NewSuite("mom", "Validates mom correctly signed")

	u := config.UInfo{
		MixName: sigFlags.mixName,
		Ver:     sigFlags.version,
		Latest:  sigFlags.latest,
	}

	mInfo, err := pkginfo.NewManifestInfo(conf, &u)
	helpers.FailIfErr(err)

	err = verifyMoM(r, mInfo.CacheLoc, mInfo.Version, sigFlags.certpath)
	helpers.FailIfErr(err)

	if r.Failed > 0 {
		os.Exit(1)
	}
}

// verifyMoM validates the MoM is signed correctly with the ca-cert Swupd_Root.pem
func verifyMoM(r *diva.Results, baseCache, version, cert string) error {
	cache := filepath.Join(baseCache, "update", version)

	mom := filepath.Join(cache, "Manifest.MoM")
	sig := filepath.Join(cache, "Manifest.MoM.sig")

	err := helpers.RunCommandSilent("openssl", "smime", "-verify", "-in", sig,
		"-inform", "der", "-content", mom, "-purpose", "any", "-CAfile", cert)

	r.Ok(err == nil, "Manifest.MoM verified")
	return err
}
