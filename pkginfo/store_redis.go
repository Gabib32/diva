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
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"

	"github.com/clearlinux/diva/bundle"
	"github.com/clearlinux/diva/internal/helpers"
	"github.com/clearlinux/mixer-tools/swupd"
	"github.com/gomodule/redigo/redis"
)

func storeIterableRedisSet(c redis.Conn, key string, value []string) error {
	for i := range value {
		if _, err := c.Do("SADD", key, value[i]); err != nil {
			return err
		}
	}
	return nil
}

// storeRepoInfoRedis stores all data in repo to the running redis-server
func storeRepoInfoRedis(c redis.Conn, repo *Repo) error {
	repoKey := fmt.Sprintf("%s%s%s", repo.Name, repo.Version, repo.Type)
	if _, err := c.Do("SET", repoKey, repo.URI); err != nil {
		return err
	}

	for i := range repo.Packages {
		if err := storeRPMInfoRedis(c, repo, repo.Packages[i]); err != nil {
			return err
		}
	}

	return nil
}

// storeRPMInfoRedis stores the rpm under the constructed repo key in the
// running redis-server
func storeRPMInfoRedis(c redis.Conn, repo *Repo, rpm *RPM) error {
	repoKey := fmt.Sprintf("%s%s%s", repo.Name, repo.Version, repo.Type)
	if _, err := c.Do("SADD", repoKey+":packages", rpm.Name); err != nil {
		return err
	}
	pkgKey := fmt.Sprintf("%s:%s", repoKey, rpm.Name)
	_, err := c.Do("HMSET", redis.Args{}.Add(pkgKey).AddFlat(rpm)...)
	if err != nil {
		return err
	}

	// store file index mapping at reponame:packagename:files
	//             filename -> fileN
	// store each file map at reponame:packagename:fileN
	//             fileN -> File{}
	fKey := pkgKey + ":files"
	for fIdx, f := range rpm.Files {
		val := fmt.Sprintf("file%d", fIdx)
		fMap := map[string]string{f.Name: val}
		_, err := c.Do("HMSET", redis.Args{}.Add(fKey).AddFlat(fMap)...)
		if err != nil {
			return err
		}
		fIdxKey := fmt.Sprintf("%s:file%d", pkgKey, fIdx)
		_, err = c.Do("HMSET", redis.Args{}.Add(fIdxKey).AddFlat(f)...)
		if err != nil {
			return err
		}
	}
	return nil
}

func storeMapAsSliceRedis(c redis.Conn, key string, val map[string]bool) error {
	valSlice, err := helpers.HashmapToSortedSlice(val)
	if err != nil {
		return err
	}
	return storeIterableRedisSet(c, key, valSlice)
}

func storeBundleInfoRedis(c redis.Conn, bundleInfo *BundleInfo, bundleset *bundle.DefinitionsSet) error {
	bundlesKey := fmt.Sprintf("%s%sbundles", bundleInfo.Name, bundleInfo.Version)

	// convert bundle definition set to slice for flat data store
	bundles := bundle.SetToSlice(*bundleset)

	// store list of all bundles
	for _, bundle := range bundles {
		_, err := c.Do("SADD", bundlesKey, bundle.Name)
		if err != nil {
			return err
		}

		// store bundle definitions
		definitionKey := fmt.Sprintf("%s:%s", bundlesKey, bundle.Name)
		_, err = c.Do("HMSET", redis.Args{}.Add(definitionKey).AddFlat(bundle)...)
		if err != nil {
			return err
		}

		// store header information for each bundle
		header := reflect.ValueOf(&bundle.Header).Elem()
		for i := 0; i < header.NumField(); i++ {
			headerKey := header.Type().Field(i).Name
			headerValue := header.Field(i).Interface()
			if _, err = c.Do("SET", definitionKey+":"+headerKey, headerValue); err != nil {
				return err
			}
		}

		if err = storeMapAsSliceRedis(c, definitionKey+":includes", bundle.Includes); err != nil {
			return err
		}
		if err = storeMapAsSliceRedis(c, definitionKey+":directPackages", bundle.DirectPackages); err != nil {
			return err
		}
		if err = storeMapAsSliceRedis(c, definitionKey+":allPackages", bundle.AllPackages); err != nil {
			return err
		}
	}
	return nil
}

func storeManifestFile(c redis.Conn, key, ftype string, files []*swupd.File) error {
	// store file index mapping at key:itemname:files
	//             filename -> fileN
	// store each file map at key:itemname:fileN
	//             fileN -> File{}
	fKey := key + ftype
	for fIdx, f := range files {
		val := fmt.Sprintf("file%d", fIdx)
		fMap := map[string]string{f.Name: val}
		_, err := c.Do("HMSET", redis.Args{}.Add(fKey).AddFlat(fMap)...)
		if err != nil {
			return err
		}

		// Encode the file struct prior to storing it in the redis database
		b := bytes.Buffer{}
		fIdxKey := fmt.Sprintf("%s:file%d", key, fIdx)
		err = gob.NewEncoder(&b).Encode(f)
		if err != nil {
			return err
		}
		_, err = c.Do("SET", fIdxKey, b.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func storeManifestHeader(c redis.Conn, header *swupd.ManifestHeader, key string) error {
	var err error
	b := bytes.Buffer{}

	// Encode the hader struct prior to storing it in the redis database
	err = gob.NewEncoder(&b).Encode(header)
	if err != nil {
		return err
	}
	_, err = c.Do("SET", fmt.Sprintf("%s:Header", key), b.Bytes())
	if err != nil {
		return err
	}

	// h := reflect.ValueOf(header).Elem()
	// for i := 0; i < h.NumField(); i++ {
	// 	headerKey := h.Type().Field(i).Name
	// 	headerValue := h.Field(i)
	// 	// the header includes fields contains a slice of manifest objects. instead
	// 	// of storing the entire object (which are already in the database), store
	// 	// a slice of all names.
	// 	if headerKey == "Includes" {
	// 		includes := []string{}
	// 		for i := 0; i < headerValue.Len(); i++ {
	// 			includes = append(includes, header.Includes[i].Name)
	// 			err := storeIterableRedisSet(c, key+":"+headerKey, includes)
	// 			if err != nil {
	// 				return err
	// 			}
	// 		}
	// 	} else {
	// 		if _, err := c.Do("SET", key+":"+headerKey, headerValue); err != nil {
	// 			return err
	// 		}
	// 	}
	// }
	return nil
}

// manifests are stored by the version they were created/changed in, not necessarily
// the version of the MoM, or the version requested.
func storeManifestRedis(c redis.Conn, mInfo *ManifestInfo, manifests []*swupd.Manifest) error {
	momKey := fmt.Sprintf("%s%smanifests", mInfo.Name, mInfo.Version)

	for _, manifest := range manifests {
		// store list of all manifest names
		_, err := c.Do("SADD", momKey, manifest.Name)
		if err != nil {
			return err
		}

		// manifest should be stored with version of that bundle
		manifestKey := fmt.Sprintf("%s%smanifests:%s", mInfo.Name, fmt.Sprint(manifest.Header.Version), manifest.Name)

		// store the entire manifest object
		_, err = c.Do("HMSET", redis.Args{}.Add(manifestKey).AddFlat(manifest)...)
		if err != nil {
			return err
		}

		// store manifest header
		err = storeManifestHeader(c, &manifest.Header, manifestKey)
		if err != nil {
			return err
		}

		// store manifest files
		err = storeManifestFile(c, manifestKey, ":Files", manifest.Files)
		if err != nil {
			return err
		}

		// store manifest deleted files
		err = storeManifestFile(c, manifestKey, ":DeletedFiles", manifest.DeletedFiles)
		if err != nil {
			return err
		}
	}

	return nil
}
