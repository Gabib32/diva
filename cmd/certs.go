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

type certsCmdFlags struct {
	mixName  string
	version  string
	latest   bool
	certpath string
}

var certFlags certsCmdFlags

func init() {
	certsCmd.Flags().StringVarP(&certFlags.mixName, "name", "n", "clear", "name of data group")
	certsCmd.Flags().StringVarP(&certFlags.version, "version", "v", "0", "version to check")
	certsCmd.Flags().BoolVar(&certFlags.latest, "latest", false, "get the latest version from upstreamURL")
	certsCmd.Flags().StringVar(&certFlags.certpath, "certpath", "/usr/share/clear/update-ca/Swupd_Root.pem", "fully qualified path to ca-cert")
}

var certsCmd = &cobra.Command{
	Use:   "certs",
	Short: "",
	Long:  ``,
	Run:   runCheckCerts,
}

func runCheckCerts(cmd *cobra.Command, args []string) {
	r := diva.NewSuite("mom", "Validates mom correctly signed")

	u := config.UInfo{
		MixName: certFlags.mixName,
		Ver:     certFlags.version,
		Latest:  certFlags.latest,
	}

	mInfo, err := pkginfo.NewManifestInfo(conf, &u)
	helpers.FailIfErr(err)

	err = verifyMoM(r, mInfo.CacheLoc, mInfo.Version, certFlags.certpath)
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
