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
	"testing"
	"time"

	"github.com/clearlinux/diva/bundle"
	"github.com/clearlinux/mixer-tools/swupd"
	"github.com/rafaeljusto/redigomock"
)

func TestGetRepoRedis(t *testing.T) {
	repo := &Repo{
		BaseInfo: BaseInfo{
			Name:    "testrepo",
			Version: "100",
		},
		Type: "B",
	}
	repoKey := fmt.Sprintf("%s%s%s", repo.Name, repo.Version, repo.Type)
	pkgsKey := fmt.Sprintf("%s:packages", repoKey)
	pkgKey := fmt.Sprintf("%s:testpkg", repoKey)
	fIdxKey := fmt.Sprintf("%s:files", pkgKey)
	fKey := fmt.Sprintf("%s:file", pkgKey)

	conn := redigomock.NewConn()
	cmds := []*redigomock.Cmd{
		conn.Command("SMEMBERS", pkgsKey).ExpectStringSlice("testpkg"),
		// this effectively tests getRPMRedis as well
		conn.Command("HGET", pkgKey, "Name").Expect("testpkg"),
		conn.Command("HGET", pkgKey, "Version").Expect("100"),
		conn.Command("HGET", pkgKey, "Release").Expect("1"),
		conn.Command("HGET", pkgKey, "Architecture").Expect("xTEST"),
		conn.Command("HGET", pkgKey, "SRPMName").Expect("testpkg.src.rpm"),
		conn.Command("HGET", pkgKey, "License").Expect("license"),
		conn.Command("HGET", pkgKey, "Requires").Expect([]byte("reqs")),
		conn.Command("HGET", pkgKey, "BuildRequires").Expect([]byte("breqs")),
		conn.Command("HGET", pkgKey, "Provides").Expect([]byte("provs")),
		// this effectively tests getFilesRedis as well
		conn.Command("HVALS", fIdxKey).ExpectStringSlice([]string{"file1", "file2"}...),
		conn.Command("HGETALL", fKey+"1").ExpectMap(map[string]string{"Name": "f1"}),
		conn.Command("HGETALL", fKey+"2").ExpectMap(map[string]string{"Name": "f2"}),
	}
	err := getRepoRedis(conn, repo)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cmds {
		if conn.Stats(c) == 0 {
			t.Errorf("expected command %s %s was not called", c.Name, c.Args)
		}
	}

	if len(repo.Packages) != 1 {
		// fatal since we access via indices below
		t.Fatalf("expected 1 package but got %d", len(repo.Packages))
	}

	p := repo.Packages[0]

	if p.Name != "testpkg" {
		t.Errorf("RPM was named '%s' but expected testpkg", p.Name)
	}

	if len(p.Files) != 2 {
		// fatal since we access via indices below
		t.Fatalf("expected 2 files but got %d", len(p.Files))
	}

	if p.Files[0].Name != "f1" {
		t.Errorf("expected file name f1 but got %s", p.Files[0].Name)
	}

	if p.Files[1].Name != "f2" {
		t.Errorf("expected file name f2 but got %s", p.Files[1].Name)
	}
}

func TestGetBundlesRedis(t *testing.T) {
	bundleInfo := &BundleInfo{}
	bundleInfo.BundleDefinitions = make(bundle.DefinitionsSet)

	bundleName := "testpkg"
	bundlesKey := fmt.Sprintf("%s%sbundles", bundleInfo.Name, bundleInfo.Version)
	bundleKey := fmt.Sprintf("%s:%s", bundlesKey, bundleName)

	conn := redigomock.NewConn()
	cmds := []*redigomock.Cmd{
		conn.Command("SMEMBERS", bundlesKey).ExpectStringSlice("testpkg"),
		conn.Command("HGET", bundleKey, "Name").Expect("testpkg"),
		conn.Command("GET", bundleKey+":Title").Expect("testpkg"),
		conn.Command("GET", bundleKey+":Description").Expect("testDesc"),
		conn.Command("GET", bundleKey+":Status").Expect("teststatus"),
		conn.Command("GET", bundleKey+":Capabilities").Expect("testcap"),
		conn.Command("GET", bundleKey+":Maintainer").Expect("testuser"),
		conn.Command("SMEMBERS", bundleKey+":includes").ExpectStringSlice("incs", "things"),
		conn.Command("SMEMBERS", bundleKey+":directPackages").ExpectStringSlice("direct packages"),
		conn.Command("SMEMBERS", bundleKey+":allPackages").ExpectStringSlice("all packages", "in", "a", "slice_yeah"),
	}

	// test single bundle
	err := getBundlesRedis(conn, bundleInfo, bundleName)
	if err != nil {
		t.Fatal(err)
	}

	for _, bun := range bundleInfo.BundleDefinitions {
		if bun.Name != bundleName {
			t.Errorf("expected bundle name %s, but got %s", bundleName, bun.Name)
		}
		if bun.Header.Capabilities != "testcap" {
			t.Errorf("expected bundle Header Capabilities testcap, but got %s", bun.Header.Capabilities)
		}
		if _, ok := bun.AllPackages["slice_yeah"]; !ok {
			t.Error("expected slice_yeah to be in AllPackages, but wasn't found")
		}
	}

	// test all bundles
	err = getBundlesRedis(conn, bundleInfo, "")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cmds {
		if conn.Stats(c) == 0 {
			t.Errorf("expected command %s %s was not called", c.Name, c.Args)
		}
	}
}

func TestGetManfiestHeaderRedis(t *testing.T) {

	// a manifests key is the <mixname><manifestversion>manifests:<manifestname>
	// the manifest version is what is stored in the manifest.MoM for each
	// individual bundle.
	manifestKey := "clear25440manifests:testname"
	conn := redigomock.NewConn()

	h := swupd.ManifestHeader{
		Format:  10,
		Version: 1,
	}

	b := bytes.Buffer{}
	err := gob.NewEncoder(&b).Encode(h)
	if err != nil {
		t.Fatal(err)
	}

	cmds := []*redigomock.Cmd{
		conn.Command("GET", manifestKey+":Header").Expect(b.Bytes()),
	}

	header, err := getManifestHeader(conn, manifestKey)
	if err != nil {
		t.Fatal(err)
	}

	// if the header object is equal to the empty header object,
	// it collected no data, so fail
	if reflect.DeepEqual(header, swupd.ManifestHeader{}) {
		t.Errorf("header is empty")
	}

	// check the timestamp parsed to be the correct object type
	if reflect.TypeOf(header.TimeStamp) != reflect.TypeOf(time.Time{}) {
		t.Errorf("header.Timestamp is not type time.Time object, but instead %v", reflect.TypeOf(header.TimeStamp))
	}

	for _, c := range cmds {
		if conn.Stats(c) == 0 {
			t.Errorf("expected command %s %s was not called", c.Name, c.Args)
		}
	}
}
