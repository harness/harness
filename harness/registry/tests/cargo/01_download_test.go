// Copyright 2023 Harness, Inc.
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

package cargoconformance

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test01Download = func() {
	ginkgo.Context("Download", func() {
		ginkgo.BeforeEach(func() {
			if !IsTestEnabled(TestDownload) {
				ginkgo.Skip("Download tests are disabled")
			}
		})

		ginkgo.It("should download an artifact", func() {
			// Upload an artifact first.
			// Use a unique package name and version for this test
			packageName := GetUniqueArtifactName("cargo", 1)
			version := GetUniqueVersion(1)

			// Generate the package payload
			body := generateCargoPackagePayload(packageName, version)

			// Set up the upload request
			req := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/new",
					TestConfig.Namespace, TestConfig.RegistryName,
				),
			)
			req.SetHeader("Content-Type", "application/crate")
			req.SetBody(body)

			// Send the request and verify the response
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(201))

			// Now download it.
			path := fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/%s/%s/download",
				TestConfig.Namespace, TestConfig.RegistryName, packageName, version,
			)
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
			gomega.Expect(string(resp.Body)).To(gomega.Equal("mock package content"))
		})

		ginkgo.It("should download an artifact from upstream", func() {
			packageName := "quote"
			version := "1.0.40"

			// get package index
			req := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/cargo/%s",
					TestConfig.Namespace, TestConfig.RegistryName,
					getIndexFilePathFromImageName(packageName),
				),
			)
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
			indexArr, err := getPackageIndexContentFromText(string(resp.Body))
			gomega.Expect(err).To(gomega.BeNil())

			// find package in indexArr with version
			var packageInfo packageIndexMetadata
			for _, pkg := range indexArr {
				if pkg.Name == packageName && pkg.Version == version {
					packageInfo = pkg
					break
				}
			}

			// check if packageInfo is empty
			if packageInfo == (packageIndexMetadata{}) {
				ginkgo.Fail("Package not found in index")
			}

			// Now download it.
			path := fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/%s/%s/download",
				TestConfig.Namespace, TestConfig.RegistryName, packageName, version,
			)
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("application/x-gzip"))
		})
	})
}
