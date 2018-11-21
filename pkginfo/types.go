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
	"strconv"

	"github.com/clearlinux/diva/bundle"
	"github.com/clearlinux/diva/internal/config"
	"github.com/clearlinux/diva/internal/helpers"
	"github.com/clearlinux/mixer-tools/swupd"
)

// BaseInfo is the base information that is more or less used by all commands
type BaseInfo struct {
	Name        string
	Version     string
	UpstreamURL string
	CacheLoc    string
}

func (b *BaseInfo) updateBaseInfo(u *config.UInfo) error {
	var err error
	if b.Version == "latest" || (b.Version == "0" && u.Latest) {
		u.Latest = true
		b.Version, err = helpers.GetLatestVersion(b.UpstreamURL)
	}
	if b.Name == "" {
		b.Name = "clear"
	}
	return err
}

func defaultBaseInfo(conf *config.Config, u *config.UInfo) BaseInfo {
	return BaseInfo{
		Name:        u.MixName,
		Version:     u.Ver,
		UpstreamURL: conf.UpstreamURL,
		CacheLoc:    conf.Paths.CacheLocation,
	}
}

// Repo defines the location, name, type, and other metadata about an RPM
// repository, a slice of pointers to RPMs, as well as an update function
// to modify the BaseInfo struct with any recent information
type Repo struct {
	BaseInfo
	URI      string
	RPMCache string
	Type     string
	Priority uint
	Packages []*RPM
}

// attempt to update the repo uri to point to the correct rpm directory, if
// neither of the two options are viable urls, do not modify the repo.URI string
func updateRepoURI(repo *Repo, loc string) {
	urls := []string{
		fmt.Sprintf("%s/%s/%s/%s", repo.URI, repo.Version, repo.Name, loc),
		fmt.Sprintf("%s/releases/%s/%s/%s", repo.URI, repo.Version, repo.Name, loc),
	}

	for _, url := range urls {
		_, err := helpers.CheckStatus(url)
		if err == nil {
			repo.URI = url
			return
		}
	}
}

func (repo *Repo) updateRepo(u *config.UInfo) error {
	err := repo.BaseInfo.updateBaseInfo(u)
	if err != nil {
		return err
	}

	if repo.Type == "" {
		repo.Type = "B"
	}

	if repo.URI == "" {
		repo.URI = repo.UpstreamURL
	}

	if repo.Type == "SRPM" {
		updateRepoURI(repo, "source/SRPMS")
	} else {
		updateRepoURI(repo, "x86_64/os")
	}
	if repo.RPMCache == config.DefaultConf().Paths.LocalRPMRepo {
		repo.RPMCache = fmt.Sprintf("%s/rpms/%s/%s/%s/packages", repo.CacheLoc, repo.Name, repo.Version, repo.Type)
	}
	return nil
}

func defaultRepo(conf *config.Config, u *config.UInfo) Repo {
	return Repo{
		BaseInfo: defaultBaseInfo(conf, u),
		RPMCache: conf.Paths.LocalRPMRepo,
		URI:      u.RepoURL,
		Type:     u.RPMType,
	}
}

// NewRepo creates a new repo object with the correct default values, and the
// updated fields depending on the UInfo/flags passed. It also contains an
// embedded BaseInfo struct
func NewRepo(conf *config.Config, u *config.UInfo) (Repo, error) {
	repo := defaultRepo(conf, u)
	return repo, repo.updateRepo(u)
}

// RPM is a packaging format that encapsulates a collection of files to install
// and assorted metadata. An RPM can be either a binary or source RPM. If
// SRPMName is empty this indicates the RPM is already a source RPM. For binary
// RPMs it will be populated with that RPMs associated source RPM name.
type RPM struct {
	Name          string
	Version       string
	Release       string
	Architecture  string
	SRPMName      string
	License       string
	Requires      []string
	BuildRequires []string
	Provides      []string
	Files         []*File
}

// File contains all information for a file in an RPM.
// Additional fields Name, Type, SwupdHash, and CurrentVersion are used by
// swupd operations.
type File struct {
	Name           string
	Type           byte
	Size           uint
	Hash           string
	SwupdHash      string
	Permissions    string
	Owner          string
	Group          string
	SymlinkTarget  string
	CurrentVersion uint
}

// BundleInfo contains information regarding bundle definitions including the
// upstream location, cached location, current repo branch, an embedded Repo
// struct, a slice of the bundle definitions, and an update function to ensure
// the configurations for the embedded structures are up to date.
type BundleInfo struct {
	BaseInfo
	BundleURL         string
	BundleCache       string
	Branch            string
	Tag               string
	BundleDefinitions bundle.DefinitionsSet
}

func (bundleInfo *BundleInfo) updateBundleInfo(u *config.UInfo) error {
	err := bundleInfo.updateBaseInfo(u)
	if err != nil {
		return err
	}

	bundleInfo.Tag = bundleInfo.Version
	// if bundle version is not a valid version number, or is 0, use latest
	_, err = strconv.Atoi(bundleInfo.Version)
	if bundleInfo.Version == "0" || err != nil {
		bundleInfo.Tag, err = helpers.GetLatestVersion(bundleInfo.UpstreamURL)
	}

	bundleInfo.BundleDefinitions = make(bundle.DefinitionsSet)
	return err
}

func defaultBundleInfo(conf *config.Config, u *config.UInfo) BundleInfo {
	return BundleInfo{
		BaseInfo:    defaultBaseInfo(conf, u),
		BundleURL:   conf.BundleDefsURL,
		BundleCache: conf.Paths.BundleDefsRepo,
	}
}

// NewBundleInfo creates a new BundleInfo instance with the correct default
// values, and the updated fields depending on the UInfo/flags passed. It
// also has an updated Repo struct embedded in it.
func NewBundleInfo(conf *config.Config, u *config.UInfo) (BundleInfo, error) {
	b := defaultBundleInfo(conf, u)
	return b, b.updateBundleInfo(u)
}

// ManifestInfo contains all fields from the BundleInfo struct, along with the
// Version as an Uint instead of a string, a minVer, and an update function for
// all embedded datatypes.
type ManifestInfo struct {
	BundleInfo
	UintVer   uint
	MinVer    uint
	MoM       *swupd.Manifest
	Manifests map[string]*swupd.Manifest
}

func (manifestInfo *ManifestInfo) updateManifestInfo(u *config.UInfo) error {
	err := manifestInfo.updateBundleInfo(u)
	if err != nil {
		return err
	}

	// store the version as an interger as well as a string
	var mv int
	mv, err = strconv.Atoi(manifestInfo.Version)
	if err != nil {
		return err
	}
	manifestInfo.UintVer = uint(mv)

	// if recursive is not passed, set the minver to be the current version
	if !u.Recursive {
		manifestInfo.MinVer = manifestInfo.UintVer
	}

	manifestInfo.Manifests = make(map[string]*swupd.Manifest)

	return nil
}

func defaultManifestInfo(conf *config.Config, u *config.UInfo) ManifestInfo {
	return ManifestInfo{
		BundleInfo: defaultBundleInfo(conf, u),
	}
}

// NewManifestInfo creates a new ManifestInfo instance and updates it with the
// associated fields.
func NewManifestInfo(conf *config.Config, u *config.UInfo) (ManifestInfo, error) {
	m := defaultManifestInfo(conf, u)
	return m, m.updateManifestInfo(u)
}
