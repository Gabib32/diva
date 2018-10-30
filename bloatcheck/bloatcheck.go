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

package bloatcheck

import (
	"fmt"
	"sort"
	"sync"

	"github.com/clearlinux/diva/pkginfo"
	"github.com/clearlinux/mixer-tools/swupd"
)

var sizeMutex sync.RWMutex

// Concurrently gets the size for each manifest
func getSizes(mInfo pkginfo.ManifestInfo, m *swupd.Manifest, bundleSizes map[string]int64) error {
	if m.Name == "os-core-update-index" {
		return nil
	}

	// Gets all includes from all bundles within BundleDefinition set
	includes, err := mInfo.BundleDefinitions.GetIncludes("")
	if err != nil {
		return err
	}
	for i := range includes {
		includeManifest, ok := mInfo.Manifests[i]
		if !ok {
			return fmt.Errorf("Unable to find manifest file %s", i)
		}
		sizeMutex.Lock()
		bundleSizes[m.Name] += int64(includeManifest.Header.ContentSize)
		sizeMutex.Unlock()
	}
	sizeMutex.Lock()
	bundleSizes[m.Name] += int64(m.Header.ContentSize)
	sizeMutex.Unlock()

	return nil
}

// GetBundleSize gets the full size of all bundles in a given version
func GetBundleSize(mInfo pkginfo.ManifestInfo) (map[string]int64, error) {
	var err error

	var wg sync.WaitGroup
	bundleWorkers := len(mInfo.Manifests)
	mChan := make(chan *swupd.Manifest)
	errChan := make(chan error, bundleWorkers)
	wg.Add(bundleWorkers)

	var bundleSizes = make(map[string]int64)

	sort.Slice(mInfo.Mom.Files, func(i, j int) bool {
		return mInfo.Mom.Files[i].Name < mInfo.Mom.Files[j].Name
	})

	for i := 0; i < bundleWorkers; i++ {
		go func() {
			defer wg.Done()
			for m := range mChan {
				// Get the total size for each bundle (without accounting for overlap between them)
				err = getSizes(mInfo, m, bundleSizes)
				if err != nil {
					errChan <- err
				}
			}
		}()
	}

	for _, m := range mInfo.Manifests {
		select {
		case mChan <- m:
		case err = <-errChan:
			break
		}
	}
	close(mChan)
	wg.Wait()

	chanLen := len(errChan)
	for i := 0; i < chanLen; i++ {
		<-errChan
	}

	return bundleSizes, err
}
