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
	"testing"
	"time"

	"github.com/clearlinux/diva/bundle"
	"github.com/clearlinux/mixer-tools/swupd"
	"github.com/rafaeljusto/redigomock"
)

func TestStoreRPMInfoRedis(t *testing.T) {
	conn := redigomock.NewConn()
	cmds := []*redigomock.Cmd{
		conn.GenericCommand("SET").Expect("ok"),
		conn.GenericCommand("SADD").Expect("ok"),
		conn.GenericCommand("HMSET").Expect("ok"),
	}

	repo := &Repo{
		BaseInfo: BaseInfo{
			Version: "100",
			Name:    "testrepo",
		},
		Type: "B",
		Packages: map[string]*RPM{
			"testpkg": {
				Name:     "testpkg",
				Version:  "100",
				Release:  "1",
				Provides: []string{"one", "two"},
				Files: []*File{
					{Name: "f1"},
					{Name: "f2"},
				},
			},
		},
	}

	if err := storeRepoInfoRedis(conn, repo); err != nil {
		t.Fatal(err)
	}

	for _, c := range cmds {
		if conn.Stats(c) == 0 {
			t.Errorf("expected command %s %s was not called", c.Name, c.Args)
		}
	}
}

func TestStoreBundleInfoRedis(t *testing.T) {
	conn := redigomock.NewConn()
	cmds := []*redigomock.Cmd{
		conn.GenericCommand("SADD").Expect("ok"),
		conn.GenericCommand("SET").Expect("ok"),
		conn.GenericCommand("HMSET").Expect("ok"),
	}

	bundleInfo := &BundleInfo{
		BaseInfo: BaseInfo{
			Name:    "clear",
			Version: "22000",
		},
	}

	BundleDefinitions := bundle.DefinitionsSet{
		"TestBundle": &bundle.Definition{
			Name: "TestBundle",
			Header: bundle.Header{
				Title: "TestBundle",
			},
		},
	}

	err := storeBundleInfoRedis(conn, bundleInfo, &BundleDefinitions)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cmds {
		if conn.Stats(c) == 0 {
			t.Errorf("expected command %s %s was not called", c.Name, c.Args)
		}
	}
}

func TestStoreManifestHeaderRedis(t *testing.T) {
	conn := redigomock.NewConn()
	cmds := []*redigomock.Cmd{
		conn.GenericCommand("SET").Expect("ok"),
	}

	header := swupd.ManifestHeader{
		Format:      uint(17),
		Version:     uint32(25520),
		Previous:    uint32(25510),
		MinVersion:  uint32(25500),
		FileCount:   uint32(4201),
		TimeStamp:   time.Now(),
		ContentSize: uint64(12345),
		Includes: []*swupd.Manifest{
			{
				Name: "bundleOne",
			},
			{
				Name: "numberTwo",
			},
			{
				Name: "three",
			},
		},
	}

	err := storeManifestHeader(conn, &header, "keymanifests")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cmds {
		if conn.Stats(c) == 0 {
			t.Errorf("expected command %s %s was not called", c.Name, c.Args)
		}
	}
}

func TestStoreManifestInfoRedis(t *testing.T) {
	conn := redigomock.NewConn()
	cmds := []*redigomock.Cmd{
		conn.GenericCommand("SADD").Expect("ok"),
		conn.GenericCommand("HMSET").Expect("ok"),
		conn.GenericCommand("SET").Expect("ok"),
	}

	mInfo := &ManifestInfo{
		BundleInfo: BundleInfo{
			BaseInfo: BaseInfo{
				Name:    "Clear",
				Version: "24000",
			},
		},
		UintVer: 22000,
		MinVer:  22000,
	}

	manifests := []*swupd.Manifest{
		{
			Name: "MoM",
			Header: swupd.ManifestHeader{
				Version: 22000,
			},
		},
		{
			Name: "bundleOne",
			Header: swupd.ManifestHeader{
				Version: 21000,
			},
		},
		{
			Name: "bundleTwo",
			Header: swupd.ManifestHeader{
				Version: 21400,
				Includes: []*swupd.Manifest{
					{
						Name: "bundleOne",
						Header: swupd.ManifestHeader{
							Version: 21000,
						},
					},
				},
			},
		},
	}

	err := storeManifestRedis(conn, mInfo, manifests)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cmds {
		if conn.Stats(c) == 0 {
			t.Errorf("expected command %s %s was not called", c.Name, c.Args)
		}
	}
}
