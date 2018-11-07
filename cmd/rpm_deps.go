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
	"fmt"
	"os"
	"strings"

	"github.com/clearlinux/diva/diva"
	"github.com/clearlinux/diva/internal/config"
	"github.com/clearlinux/diva/internal/helpers"
	"github.com/clearlinux/diva/pkginfo"
	"github.com/clearlinux/diva/rpmdata"

	"github.com/spf13/cobra"
)

type rpmDepsCmdFlags struct {
	mixName string
	version string
	latest  bool
}

var rpmDepsFlags rpmDepsCmdFlags

func init() {
	checkCmd.AddCommand(rpmDepsCmd)
	rpmDepsCmd.Flags().StringVarP(&rpmDepsFlags.mixName, "name", "n", "clear", "name of data group")
	rpmDepsCmd.Flags().StringVarP(&rpmDepsFlags.version, "version", "v", "0", "version to check")
	rpmDepsCmd.Flags().BoolVar(&rpmDepsFlags.latest, "latest", false, "get the latest version from upstreamURL")
}

var rpmDepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "",
	Long:  ``,
	Run:   runCheckDependencies,
}

func runCheckDependencies(cmd *cobra.Command, args []string) {
	u := config.UInfo{
		MixName: rpmDepsFlags.mixName,
		Ver:     rpmDepsFlags.version,
		Latest:  rpmDepsFlags.latest,
	}

	u.RPMType = "B"
	binaryRPMs, err := pkginfo.NewRepo(conf, &u)
	helpers.FailIfErr(err)

	u.RPMType = "SRPM"
	sourceRPMs, err := pkginfo.NewRepo(conf, &u)
	helpers.FailIfErr(err)

	helpers.PrintBegin("Populating repo and bundle data from database for version %s", u.Ver)
	err = pkginfo.PopulateRepo(&binaryRPMs)
	helpers.FailIfErr(err)
	err = pkginfo.PopulateRepo(&sourceRPMs)
	helpers.FailIfErr(err)

	bundleInfo, err := pkginfo.NewBundleInfo(conf, &u)
	err = pkginfo.PopulateBundles(&bundleInfo, "")
	helpers.FailIfErr(err)
	helpers.PrintComplete("Data populated successfully from database")

	// What find2.py is diong:
	// 	1. Get binary and source RPMs and their information
	// 	2. includes array rpmlib(xxxxx): a "tracking dependency" used in packaging
	// 		 to associate a package that contains/uses a feature/incompatibility with
	// 		 a version of rpmlib that can handle the feature/incompatibility.
	// 	3. Add ^ to provides dict = "__rpm_internal__"

	bundlePkgs, err := bundleInfo.BundleDefinitions.GetAllPackages("")
	if err != nil {
		fmt.Println("ERRORRR: ", err)
		// return err
	}
	//

	depInfo := rpmdata.DependencyInfo{}
	// rpmMap := rpmdata.GetRPMMappings(sourceRPMs.Packages)
	for pkg := range bundlePkgs {
		// fmt.Println(pkg)

		// faster to iterate binary repo for pkg object, or to run GetRPM for pkg??
		r, err := pkginfo.GetRPM(&binaryRPMs, pkg)
		if err != nil {
			fmt.Println("ERRORRR: ", err, pkg)
			// return err
		}

		srpmName := strings.TrimSuffix(r.SRPMName, fmt.Sprintf("-%s-%s.src.rpm", r.Version, r.Release))
		fmt.Println(srpmName)

		// if the SRPM is not found in the dependency info object, create a new
		// RPMDepInfo object to store the SRPM info, then add the RPM data
		if _, ok := depInfo[srpmName]; !ok {
			srpm, err := pkginfo.GetRPM(&sourceRPMs, srpmName)
			if err != nil {
				fmt.Println("DANG: ", err)
				// return err
			}

			depInfo[srpmName] = &rpmdata.RPMDepInfo{
				SRPM:     srpm,
				RPMs:     map[string]*pkginfo.RPM{r.Name: r},
				Provides: map[string]bool{},
				Requires: map[string]bool{},
			}
			depInfo[srpmName].RecursiveRequires(depInfo[srpmName].SRPM, &sourceRPMs, make(map[string]bool))
			depInfo[srpmName].RecursiveProvides(depInfo[srpmName].SRPM, &sourceRPMs, make(map[string]bool))

		}
		depInfo[srpmName].AddRPM(r)
		depInfo[srpmName].RecursiveRequires(r, &binaryRPMs, make(map[string]bool))
		depInfo[srpmName].RecursiveProvides(r, &binaryRPMs, make(map[string]bool))

		// }
		missing := []string{}
		for req := range depInfo[srpmName].Requires {
			fmt.Println("Req: ", req)
			if _, ok := depInfo[srpmName].Provides[req]; !ok {
				missing = append(missing, req)
			}
		}
		fmt.Println("MISSING: " + srpmName + ": " + strings.Join(missing, " || "))
		// fmt.Sprintf("%s missing deps: %s", srpmName, strings.Join(missing, " || "))
	}

	results := CheckRPMDependencies()
	if results.Failed > 0 {
		os.Exit(1)
	}
}

// CheckRPMInfo does some things...todo
func CheckRPMDependencies() *diva.Results {
	name := ""
	desc := ""
	r := diva.NewSuite(name, desc)
	r.Header(1)

	return r
}
