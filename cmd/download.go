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
var downloadFlags config.FetchingFlags

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download to cache various build information without database import",
}

var downloadAllCmd = &cobra.Command{
	Use:   "all [--version <version>] [--bundleurl|--upstreamurl <url>] [--update]",
	Run:   runDownloadAllCmd,
	Short: "Download metadata from <version> or latest available if <version> is not supplied",
	Long: `Download RPM repository, bundle definitions, and update manifests from <version>.
If <version> is not supplied, or --latest passed, download latest available content.
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

var downloadRepoCmd = &cobra.Command{
	Use:   "repo [--version <version>] [--upstreamurl <url>]",
	Run:   runDownloadRepoCmd,
	Short: "Download repo from <version> or latest if not supplied",
	Long: `Download the RPM repository at <version> from the upstream URL if <version> is
supplied, otherwise download the latest available. If --upstreamurl is supplied,
download from <url> instead of the configured/default upstream URL. The repository
is cached under the cache location defined in the configuration or default to
$HOME/clearlinux/data/rpms/<version>.`,
}

var downloadBundlesCmd = &cobra.Command{
	Use:   "bundles [--version <version>] [--bundleurl <url>]",
	Run:   runDownloadBundlesCmd,
	Short: "Download bundle definition files from <version> or latest if not passed",
	Long: `Download bundle definition files from https://github.com/clearlinux/clr-bundles or
--bundleurl <url>, if passed. The bundle definition files are cached to:
$HOME/clearlinux/projects/clr-bundles by default.`,
}

var downloadUpdateCmd = &cobra.Command{
	Use:   "update [--version <version>] [--upstreamurl <url>]",
	Run:   runDownloadUpdateCmd,
	Short: "Download the update at <version> or latest if not supplied",
	Long: `Download the update at <version> from the upstream URL if <version> is supplied,
otherwise download the latest available. If --upstreamurl is supplied, get from
<url> instead of the configured/default upstream URL. The update data is cached
under the cache location defined in the configuration or default to
$HOME/clearlinux/data/update/<version>.`,
}

var downloadCmds = []*cobra.Command{
	downloadAllCmd,
	downloadRepoCmd,
	downloadBundlesCmd,
	downloadUpdateCmd,
}

func init() {
	for _, cmd := range downloadCmds {
		downloadCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(downloadCmd)

	downloadCmd.PersistentFlags().StringVarP(&downloadFlags.MixName, "name", "n", "clear", "optional name of data group")
	downloadCmd.PersistentFlags().StringVarP(&downloadFlags.Version, "version", "v", "0", "version from which to pull data")
	downloadCmd.PersistentFlags().StringVarP(&downloadFlags.UpstreamURL, "upstreamurl", "u", "", "URL from which to pull update metadata")
	downloadCmd.PersistentFlags().BoolVar(&downloadFlags.Latest, "latest", false, "get the latest upstream version")

	downloadAllCmd.Flags().StringVarP(&downloadFlags.UpstreamRepoURL, "repourl", "m", "", "fully qualified URL from which to pull repodata")
	downloadAllCmd.Flags().StringVar(&downloadFlags.RPMCache, "rpmcache", "", "path to repo cache destination")
	downloadAllCmd.Flags().BoolVar(&downloadFlags.BinaryRPM, "binary", false, "fetches only binary RPMs")
	downloadAllCmd.Flags().BoolVar(&downloadFlags.SourceRPM, "source", false, "fetches only SRPMs")
	downloadAllCmd.Flags().StringVarP(&downloadFlags.BundleURL, "bundleurl", "b", "", "URL from which to pull bundle definitions")
	downloadAllCmd.Flags().StringVar(&downloadFlags.BundleURL, "bundlecache", "", "path to bundle cache destination")
	downloadAllCmd.Flags().BoolVar(&downloadFlags.Update, "update", false, "update pre-existing Repo data")
	downloadAllCmd.Flags().BoolVarP(&downloadFlags.Recursive, "recursive", "r", false, "recursively fetch all content referenced in update metadata")

	downloadRepoCmd.Flags().StringVarP(&downloadFlags.UpstreamRepoURL, "repourl", "m", "", "fully qualified URL from which to pull repodata")
	downloadRepoCmd.Flags().StringVar(&downloadFlags.RPMCache, "rpmcache", "", "path to repo cache destination")
	downloadRepoCmd.Flags().BoolVar(&downloadFlags.Update, "update", false, "update pre-existing Repo data")
	downloadRepoCmd.Flags().BoolVar(&downloadFlags.BinaryRPM, "binary", false, "fetches only binary RPMs")
	downloadRepoCmd.Flags().BoolVar(&downloadFlags.SourceRPM, "source", false, "fetches only SRPMs")

	downloadBundlesCmd.Flags().StringVarP(&downloadFlags.BundleURL, "bundleurl", "b", "", "URL from which to pull bundle definitions")
	downloadBundlesCmd.Flags().StringVar(&downloadFlags.BundleURL, "bundlecache", "", "path to bundle cache destination")

	downloadUpdateCmd.Flags().BoolVarP(&downloadFlags.Recursive, "recursive", "r", false, "recursively fetch all content referenced in update metadata")
}

func runDownloadAllCmd(cmd *cobra.Command, args []string) {
	runDownloadRepoCmd(cmd, args)
	runDownloadBundlesCmd(cmd, args)
	runDownloadUpdateCmd(cmd, args)
}

func runDownloadRepoCmd(cmd *cobra.Command, args []string) {
	u := config.NewUinfo(downloadFlags, conf)

	// by default download both SRPMs and binary RPMs, however allow the user to
	// pass either a "binary" or "source" flag to download only one
	if downloadFlags.BinaryRPM || (!downloadFlags.BinaryRPM && !downloadFlags.SourceRPM) {
		u.RPMType = "B"
		repo, err := pkginfo.NewRepo(conf, &u)
		helpers.FailIfErr(err)
		diva.DownloadRepo(conf, &u, &repo)
	}

	if downloadFlags.SourceRPM || (!downloadFlags.BinaryRPM && !downloadFlags.SourceRPM) {
		u.RPMType = "SRPM"
		repo, err := pkginfo.NewRepo(conf, &u)
		helpers.FailIfErr(err)
		diva.DownloadRepo(conf, &u, &repo)
	}
}

func runDownloadBundlesCmd(cmd *cobra.Command, args []string) {
	var err error
	u := config.NewUinfo(downloadFlags, conf)

	bundleInfo, err := pkginfo.NewBundleInfo(conf, &u)
	helpers.FailIfErr(err)
	diva.DownloadBundles(&bundleInfo)
}

func runDownloadUpdateCmd(cmd *cobra.Command, args []string) {
	var err error
	u := config.NewUinfo(downloadFlags, conf)

	mInfo, err := pkginfo.NewManifestInfo(conf, &u)
	helpers.FailIfErr(err)
	diva.DownloadUpdateAll(&mInfo)
}
