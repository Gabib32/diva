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
	"strings"

	"github.com/clearlinux/diva/bundle"
	"github.com/clearlinux/mixer-tools/swupd"
	"github.com/gomodule/redigo/redis"
)

func getFilesRedis(c redis.Conn, repo *Repo, p *RPM) ([]*File, error) {
	pkgKey := fmt.Sprintf("%s%s%s:%s", repo.Name, repo.Version, repo.Type, p.Name)
	fIdxsKey := fmt.Sprintf("%s:files", pkgKey)
	fIdxs, err := redis.Strings(c.Do("HVALS", fIdxsKey))
	if err != nil {
		return []*File{}, err
	}

	files := []*File{}
	for _, fIdx := range fIdxs {
		fKey := fmt.Sprintf("%s:%s", pkgKey, fIdx)
		v, err := redis.Values(c.Do("HGETALL", fKey))
		if err != nil {
			return []*File{}, err
		}

		file := &File{}
		if err = redis.ScanStruct(v, file); err != nil {
			return []*File{}, err
		}

		files = append(files, file)
	}

	return files, nil
}

func getRPMRedis(c redis.Conn, repo *Repo, rpmName string) (*RPM, error) {
	var err error
	p := &RPM{}
	pkgKey := fmt.Sprintf("%s%s%s:%s", repo.Name, repo.Version, repo.Type, rpmName)
	p.Name, err = redis.String(c.Do("HGET", pkgKey, "Name"))
	if err != nil {
		return nil, err
	}

	p.Version, err = redis.String(c.Do("HGET", pkgKey, "Version"))
	if err != nil {
		return nil, err
	}

	p.Release, err = redis.String(c.Do("HGET", pkgKey, "Release"))
	if err != nil {
		return nil, err
	}

	p.Architecture, err = redis.String(c.Do("HGET", pkgKey, "Architecture"))
	if err != nil {
		return nil, err
	}

	p.SRPMName, err = redis.String(c.Do("HGET", pkgKey, "SRPMName"))
	if err != nil {
		return nil, err
	}

	p.License, err = redis.String(c.Do("HGET", pkgKey, "License"))
	if err != nil {
		return nil, err
	}

	rb, err := redis.Bytes(c.Do("HGET", pkgKey, "Requires"))
	if err != nil {
		return nil, err
	}
	p.Requires = strings.Fields(strings.Trim(string(rb), "[]"))

	bb, err := redis.Bytes(c.Do("HGET", pkgKey, "BuildRequires"))
	if err != nil {
		return nil, err
	}
	p.BuildRequires = strings.Fields(strings.Trim(string(bb), "[]"))

	pb, err := redis.Bytes(c.Do("HGET", pkgKey, "Provides"))
	if err != nil {
		return nil, err
	}
	p.Provides = strings.Fields(strings.Trim(string(pb), "[]"))

	p.Files, err = getFilesRedis(c, repo, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// getRepoRedis retrieves all data associated with the given repo from the
// running redis-server
func getRepoRedis(c redis.Conn, repo *Repo) error {
	repoKey := fmt.Sprintf("%s%s%s", repo.Name, repo.Version, repo.Type)
	pkgsKey := fmt.Sprintf("%s:packages", repoKey)
	pIdxs, err := redis.Strings(c.Do("SMEMBERS", pkgsKey))
	if err != nil {
		return err
	}
	if len(pIdxs) == 0 {
		return fmt.Errorf(`no repo data found. Try running "diva <fetch|import> repo -v <version>" to populate database`)
	}

	for _, pn := range pIdxs {
		p, err := getRPMRedis(c, repo, pn)
		if err != nil {
			return err
		}
		repo.Packages = appendUniqueRPMName(repo.Packages, p)
	}

	return nil
}

func getBundleHeader(c redis.Conn, bundleName, bundleKey string) (bundle.Header, error) {
	var err error
	header := bundle.Header{}

	header.Title, err = redis.String(c.Do("GET", bundleKey+":Title"))
	if err != nil {
		return header, err
	}

	header.Description, err = redis.String(c.Do("GET", bundleKey+":Description"))
	if err != nil {
		return header, err
	}

	header.Status, err = redis.String(c.Do("GET", bundleKey+":Status"))
	if err != nil {
		return header, err
	}

	header.Capabilities, err = redis.String(c.Do("GET", bundleKey+":Capabilities"))
	if err != nil {
		return header, err
	}

	header.Maintainer, err = redis.String(c.Do("GET", bundleKey+":Maintainer"))
	if err != nil {
		return header, err
	}

	return header, nil
}

func getBundleRedis(c redis.Conn, bundleInfo *BundleInfo, bundleName string) error {
	var err error

	b := &bundle.Definition{
		Includes:       make(map[string]bool),
		DirectPackages: make(map[string]bool),
		AllPackages:    make(map[string]bool),
	}

	bundleKey := fmt.Sprintf("%s%sbundles:%s", bundleInfo.Name, bundleInfo.Version, bundleName)

	b.Name, err = redis.String(c.Do("HGET", bundleKey, "Name"))
	if err != nil {
		return err
	}

	b.Header, err = getBundleHeader(c, bundleName, bundleKey)
	if err != nil {
		return err
	}

	inc, err := redis.Strings(c.Do("SMEMBERS", bundleKey+":includes"))
	if err != nil {
		return err
	}

	for _, in := range inc {
		b.Includes[in] = true
	}

	dpkgs, err := redis.Strings(c.Do("SMEMBERS", bundleKey+":directPackages"))
	if err != nil {
		return err
	}

	for _, dp := range dpkgs {
		b.DirectPackages[dp] = true
	}

	apkgs, err := redis.Strings(c.Do("SMEMBERS", bundleKey+":allPackages"))
	if err != nil {
		return err
	}

	for _, ap := range apkgs {
		b.AllPackages[ap] = true
	}

	bundleInfo.BundleDefinitions[b.Name] = b

	if len(bundleInfo.BundleDefinitions) == 0 {
		return fmt.Errorf(`no bundle definitions found. Try running "diva <fetch|import> bundles -v <version>" to populate database`)
	}

	return nil
}

func getBundlesRedis(c redis.Conn, bundleInfo *BundleInfo, bundleName string) error {
	if bundleName != "" {
		return getBundleRedis(c, bundleInfo, bundleName)
	}

	bundlesKey := fmt.Sprintf("%s%sbundles", bundleInfo.Name, bundleInfo.Version)
	bIdxs, err := redis.Strings(c.Do("SMEMBERS", bundlesKey))
	if err != nil {
		return err
	}

	if len(bIdxs) == 0 {
		return fmt.Errorf(`no bundle definitions found. Try running "diva <fetch|import> bundles -v <version>" to populate database`)
	}

	for _, bn := range bIdxs {
		err := getBundleRedis(c, bundleInfo, bn)
		if err != nil {
			return err
		}
	}

	return nil
}

func getManifestHeader(c redis.Conn, key string) (swupd.ManifestHeader, error) {
	var err error
	header := swupd.ManifestHeader{}

	v, err := redis.Bytes(c.Do("GET", key+":Header"))
	if err != nil {
		return header, err
	}

	err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(&header)
	if err != nil {
		return header, err
	}

	// var parsed uint64
	// header := swupd.ManifestHeader{}
	//
	// h := reflect.ValueOf(&header).Elem()
	// for i := 0; i < h.NumField(); i++ {
	// 	k := h.Type().Field(i).Name
	//
	// 	// vt is the type of struct value to conver byte slice to
	// 	vt := h.Field(i).Type().Kind()
	//
	// 	var s string
	// 	var err error
	// 	if vt != reflect.Slice {
	// 		s, err = redis.String(c.Do("GET", fmt.Sprintf("%s:%s", key, k)))
	// 		if err != nil {
	// 			return header, err
	// 		}
	// 	}
	//
	// 	switch vt {
	// 	case reflect.Uint:
	// 		if parsed, err = strconv.ParseUint(s, 10, 8); err != nil {
	// 			return header, err
	// 		}
	// 		h.Field(i).SetUint(parsed)
	// 	case reflect.Uint32:
	// 		if parsed, err = strconv.ParseUint(s, 10, 32); err != nil {
	// 			return header, err
	// 		}
	// 		h.Field(i).SetUint(parsed)
	// 	case reflect.Struct:
	// 		// check that the struct is the time.Time object
	// 		if h.Field(i).Type() == reflect.TypeOf(time.Time{}) {
	// 			t, err := time.Parse("2006-01-02 15:04:05 -0700 PDT", s)
	// 			if err != nil {
	// 				return header, err
	// 			}
	// 			h.Field(i).Set(reflect.ValueOf(t))
	// 		}
	// 	case reflect.Slice:
	// 		if h.Field(i).Type() == reflect.TypeOf([]*swupd.Manifest{}) {
	// 			incs, err := redis.Strings(c.Do("SMEMBERS", fmt.Sprintf("%s:%s", key, k)))
	// 			if err != nil {
	// 				return header, err
	// 			}
	// 			manifests := []*swupd.Manifest{}
	// 			for _, in := range incs {
	// 				manifests = append(manifests, &swupd.Manifest{Name: in})
	// 			}
	// 			h.Field(i).Set(reflect.ValueOf(manifests))
	// 		}
	// 	}
	// }
	return header, nil
}

func getManifestFilesRedis(c redis.Conn, manifestKey, filesKey string) ([]*swupd.File, error) {
	fIdxsKey := fmt.Sprintf("%s:%s", manifestKey, filesKey)
	fIdxs, err := redis.Strings(c.Do("HVALS", fIdxsKey))
	if err != nil {
		return []*swupd.File{}, err
	}

	files := []*swupd.File{}
	for _, fIdx := range fIdxs {
		fKey := fmt.Sprintf("%s:%s", manifestKey, fIdx)
		v, err := redis.Bytes(c.Do("GET", fKey))
		if err != nil {
			return []*swupd.File{}, err
		}

		// decode file and store into swupd file object
		file := &swupd.File{}
		err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(file)
		if err != nil {
			return []*swupd.File{}, err
		}
		files = append(files, file)
	}
	return files, nil
}

func getManifestRedis(c redis.Conn, manifestKey string) (*swupd.Manifest, error) {
	var err error
	manifest := swupd.Manifest{}

	manifest.Name, err = redis.String(c.Do("HGET", manifestKey, "Name"))
	if err != nil {
		return &manifest, err
	}

	manifest.Header, err = getManifestHeader(c, manifestKey)
	if err != nil {
		return &manifest, err
	}

	manifest.Files, err = getManifestFilesRedis(c, manifestKey, "Files")
	if err != nil {
		return &manifest, err
	}

	manifest.DeletedFiles, err = getManifestFilesRedis(c, manifestKey, "DeletedFiles")
	if err != nil {
		return &manifest, err
	}

	return &manifest, nil
}

func getManifestsRedis(c redis.Conn, mInfo *ManifestInfo) error {
	mIdxs, err := redis.Strings(c.Do("SMEMBERS", fmt.Sprintf("%s%smanifests", mInfo.Name, mInfo.Version)))
	if err != nil {
		return err
	}

	if len(mIdxs) == 0 {
		return fmt.Errorf(`no manifests found. Try running "diva <fetch|import> update -v <version>" to populate database`)
	}

	momKey := fmt.Sprintf("%s%smanifests:MoM", mInfo.Name, mInfo.Version)
	mInfo.Mom, err = getManifestRedis(c, momKey)
	if err != nil {
		return err
	}

	for _, mFile := range mInfo.Mom.Files {
		manifestKey := fmt.Sprintf("%s%smanifests:%s", mInfo.Name, fmt.Sprint(mFile.Version), mFile.Name)
		m, err := getManifestRedis(c, manifestKey)
		if err != nil {
			return err
		}
		mInfo.Manifests[m.Name] = m
	}

	return nil
}
