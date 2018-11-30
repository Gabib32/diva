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
	"fmt"
	"os"

	"github.com/cavaliercoder/go-rpm"
	"github.com/gomodule/redigo/redis"
)

// ImportAllRPMs imports all RPMs from a given repository. It populates the
// passed repo with all RPMs imported.
func ImportAllRPMs(repo *Repo, update bool) error {
	var err error

	if err = loadRepoFromCache(repo, repo.RPMCache); err != nil {
		return err
	}

	var c redis.Conn
	if c, err = initRedis(0); err != nil {
		return err
	}
	defer func() {
		_ = c.Close()
	}()

	return storeRepoInfoRedis(c, repo)
}

// ImportRPM imports a single RPM named <rpm> from a given repo. It adds the
// RPM to the passed repo and returns the RPM struct.
func ImportRPM(repo *Repo, rpm, path string, update bool) (*RPM, error) {
	var err error

	if err = loadRepoFromCache(repo, path); err != nil {
		return nil, err
	}

	var c redis.Conn
	if c, err = initRedis(0); err != nil {
		return nil, err
	}
	defer func() {
		_ = c.Close()
	}()
	for _, r := range repo.Packages {
		if r.Name == rpm {
			return r, storeRPMInfoRedis(c, repo, r)
		}
	}

	return nil, fmt.Errorf("unable to find %s RPM in %s repo", rpm, repo.Name)
}

func fileFromPackageFile(pkgFI *rpm.FileInfo) *File {
	var t byte
	switch mode := pkgFI.Mode(); {
	case mode.IsRegular():
		t = byte('F')
	case pkgFI.IsDir():
		t = byte('D')
	case mode&os.ModeSymlink != 0:
		t = byte('L')
	default:
		// unrecognized file type, what do we want to do here?
		t = byte('F')
	}

	return &File{
		Name:           pkgFI.Name(),
		Type:           t,
		Size:           uint(pkgFI.Size()),
		Hash:           pkgFI.Digest(),
		SwupdHash:      "", // TODO
		Permissions:    pkgFI.Mode().Perm().String(),
		Owner:          pkgFI.Owner(),
		Group:          pkgFI.Group(),
		SymlinkTarget:  pkgFI.Linkname(),
		CurrentVersion: 0,
	}
}

func rpmFromPackage(pkg *rpm.PackageFile) *RPM {
	rpm := &RPM{
		Name:         pkg.Name(),
		Version:      pkg.Version(),
		Release:      pkg.Release(),
		Architecture: pkg.Architecture(),
		SRPMName:     pkg.SourceRPM(),
		License:      pkg.License(),
	}

	for _, d := range pkg.Requires() {
		rpm.Requires = append(rpm.Requires, d.Name())
	}

	for _, p := range pkg.Provides() {
		rpm.Provides = append(rpm.Provides, p.Name())
	}

	for _, f := range pkg.Files() {
		rpm.Files = append(rpm.Files, fileFromPackageFile(&f))
	}

	return rpm
}

func appendUniqueRPMName(rpms []*RPM, rpm *RPM) []*RPM {
	for i := range rpms {
		if rpms[i].Name == rpm.Name {
			return rpms
		}
	}

	return append(rpms, rpm)
}

func loadRepoFromCache(repo *Repo, cacheLoc string) error {
	rpms, err := rpm.OpenPackageFiles(cacheLoc)
	if err != nil {
		return err
	}

	for i := range rpms {
		repo.Packages = appendUniqueRPMName(repo.Packages, rpmFromPackage(rpms[i]))
	}

	return nil
}
