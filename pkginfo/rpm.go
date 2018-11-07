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

package pkginfo

import (
	"github.com/gomodule/redigo/redis"
)

// getRPMFromRepo returns a pointer to the RPM that matches the rpm name. If
// the repo does not contain the rpm, returns nil
func getRPMFromRepo(repo *Repo, rpm string) *RPM {
	for _, r := range repo.Packages {
		if r.Name == rpm {
			return r
		}
	}
	return nil
}

// func GetRPMFromRepo(repo *Repo, rpm string) (*RPM, error) {
// 	sort.Slice(repo.Packages, func(i int, j int) bool {
// 		return repo.Packages[i].Name < repo.Packages[j].Name
// 	})
//
// 	return nil, nil
// }

// GetRPM fetches information about an RPM in a repo. Returns a pointer to the
// associated RPM struct.
func GetRPM(repo *Repo, rpm string) (*RPM, error) {
	// if r := getRPMFromRepo(repo, rpm); r != nil {
	// 	return r, nil
	// }

	r := repo.Packages[rpm]
	if r != nil {
		return r, nil
	}

	var c redis.Conn
	var err error
	if c, err = initRedis(0); err != nil {
		return nil, err
	}
	defer func() {
		_ = c.Close()
	}()
	return getRPMRedis(c, repo, rpm)
}

// GetSRPMName returns the SRPMName field of the given rpm. The rpm specified
// must be a binary or debuginfo RPM. If it is a source RPM or the RPM does not
// exist in the Repo, an error is returned.
func GetSRPMName(repo *Repo, rpm string) (string, error) {
	r, err := GetRPM(repo, rpm)
	if err != nil {
		return "", err
	}

	return r.SRPMName, nil
}

// GetRequires gets all runtime requirements for the given RPM. If the RPM is a
// source RPM an error is returned.
func GetRequires(repo *Repo, rpm string) ([]string, error) {
	r, err := GetRPM(repo, rpm)
	if err != nil {
		return []string{}, nil
	}

	return r.Requires, nil
}

// GetBuildRequires gets all build requirements for the given source RPM. If
// the RPM is a binary or debuginfo RPM an error is returned.
func GetBuildRequires(repo *Repo, rpm string) ([]string, error) {
	r, err := GetRPM(repo, rpm)
	if err != nil {
		return []string{}, nil
	}

	return r.BuildRequires, nil
}

// GetProvides gets all symbols provided by the given RPM.
func GetProvides(repo *Repo, rpm string) ([]string, error) {
	r, err := GetRPM(repo, rpm)
	if err != nil {
		return []string{}, nil
	}

	return r.Provides, nil
}

// GetFiles gets the complete slice of files that are installed by the given
// RPM.
func GetFiles(repo *Repo, rpm string) ([]*File, error) {
	r, err := GetRPM(repo, rpm)
	if err != nil {
		return []*File{}, nil
	}

	return r.Files, nil
}

// GetFileNames gets the complete slice of filenames that are installed by the
// given RPM.
func GetFileNames(repo *Repo, rpm string) ([]string, error) {
	fs, err := GetFiles(repo, rpm)
	if err != nil {
		return []string{}, err
	}

	fNames := []string{}
	for i := range fs {
		fNames = append(fNames, fs[i].Name)
	}

	return fNames, nil
}
