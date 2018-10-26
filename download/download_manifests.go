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

package download

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/clearlinux/diva/internal/helpers"
	"github.com/clearlinux/diva/pkginfo"
	"github.com/clearlinux/mixer-tools/swupd"
)

// GetManfest downloads a manifest to outF
func GetManfest(baseURL string, version string, component, outF string) error {
	if _, err := os.Lstat(outF); err == nil {
		return nil
	}
	url := fmt.Sprintf("%s/update/%s/Manifest.%s.tar", baseURL, version, component)

	err := os.MkdirAll(filepath.Dir(outF), 0744)
	if err != nil {
		return err
	}
	err = helpers.TarExtractURL(url, outF)
	if err != nil {
		return err
	}

	return nil
}

// GetMom returns the downloaded and parsed swupd.manifest mom object
func GetMom(mInfo pkginfo.ManifestInfo) (*swupd.Manifest, error) {
	baseCache := filepath.Join(mInfo.CacheLoc, "update")
	outMoM := filepath.Join(baseCache, mInfo.Version, "Manifest.MoM")
	err := GetManfest(mInfo.UpstreamURL, mInfo.Version, "MoM", outMoM)
	if err != nil {
		return nil, err
	}
	return swupd.ParseManifestFile(outMoM)
}

// UpdateContent downloads all manifests from the MOM file
func UpdateContent(mInfo pkginfo.ManifestInfo) error {
	mom, err := GetMom(mInfo)
	if err != nil {
		return err
	}
	baseCache := filepath.Join(mInfo.CacheLoc, "update")

	// iterate mom files and download all manifests to cache location based on ver
	for i := range mom.Files {
		ver := fmt.Sprint(mom.Files[i].Version)
		outMan := filepath.Join(baseCache, ver, "Manifest."+mom.Files[i].Name)
		err := GetManfest(mInfo.UpstreamURL, ver, mom.Files[i].Name, outMan)
		if err != nil {
			return err
		}
	}
	return nil
}

type finfo struct {
	out string
	url string
	err error
}

func getAllManifestFiles(mInfo pkginfo.ManifestInfo) (map[string]finfo, error) {
	dlFiles := make(map[string]finfo)
	baseCache := filepath.Join(mInfo.CacheLoc, "update")

	mom, err := GetMom(mInfo)
	if err != nil {
		return nil, err
	}

	// this is fast, no need to parallelize
	for i := range mom.Files {
		mv := fmt.Sprint(mom.Files[i].Version)
		outMan := filepath.Join(baseCache, mv, "Manifest."+mom.Files[i].Name)
		err := GetManfest(mInfo.UpstreamURL, mv, mom.Files[i].Name, outMan)
		if err != nil {
			return nil, err
		}

		m, err := swupd.ParseManifestFile(outMan)
		if err != nil {
			return nil, err
		}

		for _, f := range m.Files {
			// TODO: What is this and why does it cause wg.Wait() to deadlock
			// Also, will this not download files less than MinVer (which is current version unless recursion is passed)
			// Do we need an actual minversion that is the earliest minversion to format bump
			// if uint(f.Version) < mInfo.MinVer ||
			if !f.Present() {
				continue
			}

			fURL := fmt.Sprintf("%s/update/%d/files/%s.tar", mInfo.UpstreamURL, f.Version, f.Hash)
			fOut := filepath.Join(baseCache, fmt.Sprint(f.Version), "files", f.Hash.String()+".tar")
			fi := finfo{out: fOut, url: fURL}
			dlFiles[fOut] = fi
		}
	}
	return dlFiles, nil
}

// UpdateFiles downloads relevant files for u.Ver from u.URL
func UpdateFiles(mInfo *pkginfo.ManifestInfo) error {
	dlFiles, err := getAllManifestFiles(*mInfo)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	nworkers := 8
	wg.Add(nworkers)
	fChan := make(chan finfo)
	errChan := make(chan error, nworkers)

	for i := 0; i < nworkers; i++ {
		go func() {
			defer wg.Done()
			for f := range fChan {
				// we already have this file cached
				if _, err := os.Lstat(strings.TrimSuffix(f.out, ".tar")); err == nil {
					continue
				}

				f.err = helpers.TarExtractURL(f.url, f.out)
				_ = os.Remove(f.out)

				if f.err != nil {
					errChan <- f.err
				}
			}
		}()
	}

	for f := range dlFiles {
		fChan <- dlFiles[f]
	}
	close(fChan)
	wg.Wait()

	if len(errChan) > 0 {
		helpers.PrintComplete("errors downloading %d files", len(errChan))
	}
	return nil
}
