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
	"os"

	"github.com/clearlinux/diva/diva"
	"github.com/clearlinux/diva/internal/config"
	"github.com/clearlinux/diva/internal/helpers"
	"github.com/clearlinux/diva/pkginfo"
	"github.com/clearlinux/diva/updatecontent"

	"github.com/spf13/cobra"
)

func init() {
	ucCmd.Flags().StringVarP(&ucFlags.mixName, "name", "n", "clear", "name of data group")
	ucCmd.Flags().StringVarP(&ucFlags.version, "version", "v", "0", "version to check")
	ucCmd.Flags().BoolVar(&ucFlags.latest, "latest", false, "get the latest version from upstreamURL")
	ucCmd.Flags().BoolVarP(&ucFlags.recursive, "recursive", "r", false, "perform complete recursive check")
}

type ucCmdFlags struct {
	mixName   string
	version   string
	latest    bool
	recursive bool
}

var ucFlags ucCmdFlags

var ucCmd = &cobra.Command{
	Use:   "updatecontent",
	Short: "Validate update file and pack content",
	Long: `Validate update content for <version> or latest if --latest is passed.
Validates that all file and pack content is available and correct and their
hashes match those provided in their respective manifests. If --recursive was
passed, perform the check on all update content reachable through the
manifests, otherwise validate only the current version.`,
	Run: runUCCheck,
}

func runUCCheck(cmd *cobra.Command, args []string) {
	u := config.UInfo{
		MixName:   ucFlags.mixName,
		Ver:       ucFlags.version,
		Latest:    ucFlags.latest,
		Recursive: ucFlags.recursive,
	}

	manifestInfo, err := pkginfo.NewManifestInfo(conf, &u)
	helpers.FailIfErr(err)

	helpers.PrintBegin("Populating manifests from database")
	err = pkginfo.PopulateManifests(&manifestInfo)
	helpers.FailIfErr(err)
	helpers.PrintComplete("Manifests populated")

	results, err := UCCheck(&manifestInfo)
	helpers.FailIfErr(err)

	if results.Failed > 0 {
		os.Exit(1)
	}
}

// UCCheck runs update content checks against manifests and their related file
// and pack contents
func UCCheck(m *pkginfo.ManifestInfo) (*diva.Results, error) {
	var err error
	r := diva.NewSuite("updatecontent", "check update content for release")

	r.Header(0)
	err = updatecontent.CheckManifestHashes(r, conf, m.UintVer, m.MinVer)
	if err != nil {
		return r, err
	}
	err = updatecontent.CheckFileHashes(r, conf, m.UintVer, m.MinVer)
	if err != nil {
		return r, err
	}
	err = updatecontent.CheckPacks(r, conf, m.UintVer, m.MinVer, true)
	if err != nil {
		return r, err
	}
	err = updatecontent.CheckPacks(r, conf, m.UintVer, m.MinVer, false)
	if err != nil {
		return r, err
	}
	return r, err
}
