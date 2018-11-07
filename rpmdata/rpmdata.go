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

package rpmdata

import (
	"github.com/clearlinux/diva/pkginfo"
)

type rpmMapping map[string][]*pkginfo.RPM

func GetRPMMappings(rpms []*pkginfo.RPM) rpmMapping {
	rpmMap := make(rpmMapping)
	for _, r := range rpms {
		if _, ok := rpmMap[r.SRPMName]; ok {
			rpmMap[r.SRPMName] = append(rpmMap[r.SRPMName], r)
		} else {
			rpmMap[r.SRPMName] = []*pkginfo.RPM{r}
		}
	}
	return rpmMap
}

type DependencyInfo map[string]*RPMDepInfo

type RPMDepInfo struct {
	SRPM        *pkginfo.RPM
	RPMs        map[string]*pkginfo.RPM
	Provides    map[string]bool
	Requires    map[string]bool
	RevProvides map[string]bool
	//packageFiles    map[string]string
	//revPackageFiles map[string]string
}

func (rpmDep *RPMDepInfo) AddRPM(r *pkginfo.RPM) {
	rpmDep.RPMs[r.Name] = r
}

func (rpmDep *RPMDepInfo) RecursiveRequires(r *pkginfo.RPM, repo *pkginfo.Repo, visited map[string]bool) {
	// fmt.Println("Package: ", r.Name)
	if _, ok := visited[r.Name]; ok {
		return
	}
	visited[r.Name] = true
	for _, i := range r.Requires {
		rpmDep.Requires[i] = true
		if dep, err := pkginfo.GetRPM(repo, i); err == nil {
			rpmDep.RecursiveRequires(dep, repo, visited)
		}
	}
}

func (rpmDep *RPMDepInfo) RecursiveProvides(r *pkginfo.RPM, repo *pkginfo.Repo, visited map[string]bool) {
	if _, ok := visited[r.Name]; ok {
		return
	}
	visited[r.Name] = true
	for _, i := range r.Provides {
		rpmDep.Provides[i] = true
		if dep, err := pkginfo.GetRPM(repo, i); err == nil {
			rpmDep.RecursiveProvides(dep, repo, visited)
		}
	}
}
