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

// PopulateRepo populates the repo struct with all RPMs from the database
func PopulateRepo(repo *Repo) error {
	var err error
	var c redis.Conn
	if c, err = initRedis(0); err != nil {
		return err
	}
	defer func() {
		_ = c.Close()
	}()

	return getRepoRedis(c, repo)
}

// PopulateBundles populates BundleInfo with bundle definitions from the database
func PopulateBundles(bundleInfo *BundleInfo, bundleName string) error {
	var err error
	var c redis.Conn
	if c, err = initRedis(0); err != nil {
		return err
	}
	defer func() {
		_ = c.Close()
	}()

	return getBundlesRedis(c, bundleInfo, bundleName)
}

// PopulateManifests queries the database for manifest information and stores
// it into the mInfo object
func PopulateManifests(mInfo *ManifestInfo) error {
	var err error
	var c redis.Conn
	if c, err = initRedis(0); err != nil {
		return err
	}
	defer func() {
		_ = c.Close()
	}()

	return getManifestsRedis(c, mInfo)
}
