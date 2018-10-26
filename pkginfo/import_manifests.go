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
	"path/filepath"

	"github.com/clearlinux/mixer-tools/swupd"
	"github.com/gomodule/redigo/redis"
)

func getMoM(bundleInfo BundleInfo) (*swupd.Manifest, error) {
	baseCache := filepath.Join(bundleInfo.CacheLoc, "update")
	momPath := filepath.Join(baseCache, bundleInfo.Version, "Manifest.MoM")
	return swupd.ParseManifestFile(momPath)
}

func getManifests(bundleInfo BundleInfo) ([]*swupd.Manifest, error) {
	var path string
	baseCache := filepath.Join(bundleInfo.CacheLoc, "update")

	mom, err := getMoM(bundleInfo)
	if err != nil {
		return nil, err
	}

	// Add mom to manifests slice
	manifests := []*swupd.Manifest{mom}
	for _, manifest := range mom.Files {
		path = filepath.Join(baseCache, fmt.Sprint(manifest.Version), "Manifest."+manifest.Name)
		mf, err := swupd.ParseManifestFile(path)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, mf)
	}
	return manifests, nil
}

// ImportManifests gets the manifests from their cached location and stores
// them into the database
func ImportManifests(mInfo *ManifestInfo) error {
	manifests, err := getManifests(mInfo.BundleInfo)
	if err != nil {
		return err
	}

	var c redis.Conn
	if c, err = initRedis(0); err != nil {
		return err
	}
	defer func() {
		_ = c.Close()
	}()

	return storeManifestRedis(c, mInfo, manifests)
}
