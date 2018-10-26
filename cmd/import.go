// Copyright © 2018 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/clearlinux/diva/diva"
	"github.com/clearlinux/diva/internal/config"
	"github.com/clearlinux/diva/internal/helpers"
	"github.com/clearlinux/diva/pkginfo"
	"github.com/spf13/cobra"
)

// the FetchingFlags struct can be found in the config file
var importFlags config.FetchingFlags

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "import to cache various build information without database import",
}

var importAllCmd = &cobra.Command{
	Use:   "all [--version <version>] [--bundleurl|--upstreamurl <url>] [--update]",
	Run:   runImportAllCmd,
	Short: "Import metadata from <version> or latest available if <version> is not supplied",
	Long: `Import RPM repository, bundle definitions, and update manifests from <version>.
If <version> is not supplied, or --latest passed, import latest available content.
Pass --upstreamurl to cache RPMs and update metadata from a location other than the
configured Upstream URL and --bundleurl to fetch the bundle definitions from a
location other than default. If --update is passed, the cached data will be
updated with new information from the upstream url.

RPMs will be cached under the cache location defined in your configuration or
default to $HOME/clearlinux/data/rpms/<version>. Bundle definition files will
be placed at $HOME/clearlinux/projects/clr-bundles by default. Update metadata
will be cached under the cache location as well, defaulting to
$HOME/clearlinux/data/update/<version>.`,
}

var importRepoCmd = &cobra.Command{
	Use:   "repo [--version <version>] [--upstreamurl <url>]",
	Run:   runImportRepoCmd,
	Short: "Import repo from <version> or latest if not supplied",
	Long: `Import the RPM repository at <version> from the upstream URL if <version> is
supplied, otherwise import the latest available. If --upstreamurl is supplied,
import from <url> instead of the configured/default upstream URL. The repository
is cached under the cache location defined in the configuration or default to
$HOME/clearlinux/data/rpms/<version>.`,
}

var importBundlesCmd = &cobra.Command{
	Use:   "bundles [--version <version>] [--bundleurl <url>]",
	Run:   runImportBundlesCmd,
	Short: "Import bundle definition files from <version> or latest if not passed",
	Long: `Import bundle definition files from https://github.com/clearlinux/clr-bundles or
--bundleurl <url>, if passed. The bundle definition files are cached to:
$HOME/clearlinux/projects/clr-bundles by default.`,
}

var importUpdateCmd = &cobra.Command{
	Use:   "update [--version <version>] [--upstreamurl <url>]",
	Run:   runImportUpdateCmd,
	Short: "Import the update at <version> or latest if not supplied",
	Long: `Import the update at <version> from the upstream URL if <version> is supplied,
otherwise import the latest available. If --upstreamurl is supplied, get from
<url> instead of the configured/default upstream URL. The update data is cached
under the cache location defined in the configuration or default to
$HOME/clearlinux/data/update/<version>.`,
}

var importCmds = []*cobra.Command{
	importAllCmd,
	importRepoCmd,
	importBundlesCmd,
	importUpdateCmd,
}

func init() {
	for _, cmd := range importCmds {
		importCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(importCmd)

	importCmd.PersistentFlags().StringVarP(&importFlags.MixName, "name", "n", "clear", "optional name of data group")
	importCmd.PersistentFlags().StringVarP(&importFlags.Version, "version", "v", "0", "version from which to pull data")
	importCmd.PersistentFlags().StringVarP(&importFlags.UpstreamURL, "upstreamurl", "u", "", "URL from which to pull update metadata")
	importCmd.PersistentFlags().BoolVar(&importFlags.Latest, "latest", false, "get the latest upstream version")

	importAllCmd.Flags().StringVarP(&importFlags.UpstreamRepoURL, "repourl", "m", "", "fully qualified URL from which to pull repodata")
	importAllCmd.Flags().StringVar(&importFlags.RPMCache, "rpmcache", "", "path to repo cache destination")
	importAllCmd.Flags().BoolVar(&importFlags.BinaryRPM, "binary", false, "fetches only binary RPMs")
	importAllCmd.Flags().BoolVar(&importFlags.SourceRPM, "source", false, "fetches only SRPMs")
	importAllCmd.Flags().StringVarP(&importFlags.BundleURL, "bundleurl", "b", "", "URL from which to pull bundle definitions")
	importAllCmd.Flags().StringVar(&importFlags.BundleURL, "bundlecache", "", "path to bundle cache destination")
	importAllCmd.Flags().BoolVar(&importFlags.Update, "update", false, "update pre-existing Repo data")
	importAllCmd.Flags().BoolVarP(&importFlags.Recursive, "recursive", "r", false, "recursively fetch all content referenced in update metadata")

	importRepoCmd.Flags().StringVarP(&importFlags.UpstreamRepoURL, "repourl", "m", "", "fully qualified URL from which to pull repodata")
	importRepoCmd.Flags().StringVar(&importFlags.RPMCache, "rpmcache", "", "path to repo cache destination")
	importRepoCmd.Flags().BoolVar(&importFlags.BinaryRPM, "binary", false, "fetches only binary RPMs")
	importRepoCmd.Flags().BoolVar(&importFlags.SourceRPM, "source", false, "fetches only SRPMs")

	importBundlesCmd.Flags().StringVarP(&importFlags.BundleURL, "bundleurl", "b", "", "URL from which to pull bundle definitions")
	importBundlesCmd.Flags().StringVar(&importFlags.BundleURL, "bundlecache", "", "path to bundle cache destination")

	importUpdateCmd.Flags().BoolVarP(&importFlags.Recursive, "recursive", "r", false, "recursively fetch all content referenced in update metadata")
}

func runImportAllCmd(cmd *cobra.Command, args []string) {
	runImportRepoCmd(cmd, args)
	runImportBundlesCmd(cmd, args)
	runImportUpdateCmd(cmd, args)
}

func runImportRepoCmd(cmd *cobra.Command, args []string) {
	u := config.NewUinfo(importFlags, conf)

	// by default import both SRPMs and binary RPMs, however allow the user to
	// pass either a "binary" or "source" flag to import only one
	if importFlags.BinaryRPM || (!importFlags.BinaryRPM && !importFlags.SourceRPM) {
		u.RPMType = "B"
		repo, err := pkginfo.NewRepo(conf, &u)
		helpers.FailIfErr(err)
		diva.ImportRepo(conf, &u, &repo)
	}
	if importFlags.SourceRPM || (!importFlags.BinaryRPM && !importFlags.SourceRPM) {
		u.RPMType = "SRPM"
		repo, err := pkginfo.NewRepo(conf, &u)
		helpers.FailIfErr(err)
		diva.ImportRepo(conf, &u, &repo)
	}
}

func runImportBundlesCmd(cmd *cobra.Command, args []string) {
	u := config.NewUinfo(importFlags, conf)
	bundleInfo, err := pkginfo.NewBundleInfo(conf, &u)
	helpers.FailIfErr(err)
	diva.ImportBundles(&bundleInfo)
}

func runImportUpdateCmd(cmd *cobra.Command, args []string) {
	u := config.NewUinfo(importFlags, conf)
	mInfo, err := pkginfo.NewManifestInfo(conf, &u)
	helpers.FailIfErr(err)
	diva.ImportUpdate(&mInfo)
}
