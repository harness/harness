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

// registry/tests/cargo/02_upload_test.go
package cargoconformance

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test03UpdateYank = func() {
	ginkgo.Context("Update Yank", func() {
		var packageName string
		var version string
		ginkgo.BeforeEach(func() {
			if !IsTestEnabled(TestUpdateYank) {
				ginkgo.Skip("Update Yank tests are disabled")
			}
		})

		ginkgo.It("should update yank to true for a Cargo package", func() {
			// Use a unique package name and version for this test
			packageName = GetUniqueArtifactName("cargo", 3)
			version = GetUniqueVersion(3)

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

			// get package index
			var indexArr []packageIndexMetadata
			gomega.Eventually(func() int {
				req = client.NewRequest(
					"GET",
					fmt.Sprintf("/pkg/%s/%s/cargo/%s",
						TestConfig.Namespace, TestConfig.RegistryName,
						getIndexFilePathFromImageName(packageName),
					),
				)
				resp, err = client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
				indexArr, err = getPackageIndexContentFromText(string(resp.Body))
				gomega.Expect(err).To(gomega.BeNil())
				return len(indexArr)
			}, 5, 1).Should(gomega.Equal(1))

			gomega.Expect(indexArr[0].Name).To(gomega.Equal(packageName))
			gomega.Expect(indexArr[0].Version).To(gomega.Equal(version))
			gomega.Expect(indexArr[0].Yanked).To(gomega.Equal(false))

			// update yank to true
			req = client.NewRequest(
				"DELETE",
				fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/%s/%s/yank",
					TestConfig.Namespace, TestConfig.RegistryName,
					packageName, version,
				),
			)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))

			// get package index
			gomega.Eventually(func() bool {
				req = client.NewRequest(
					"GET",
					fmt.Sprintf("/pkg/%s/%s/cargo/%s",
						TestConfig.Namespace, TestConfig.RegistryName,
						getIndexFilePathFromImageName(packageName),
					),
				)
				resp, err = client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
				indexArr, err = getPackageIndexContentFromText(string(resp.Body))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(len(indexArr)).To(gomega.Equal(1))
				gomega.Expect(indexArr[0].Name).To(gomega.Equal(packageName))
				gomega.Expect(indexArr[0].Version).To(gomega.Equal(version))
				return indexArr[0].Yanked
			}, 5, 1).Should(gomega.Equal(true))
		})

		ginkgo.It("should update yank to false for a Cargo package", func() {
			// update yank to false
			req := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/%s/%s/unyank",
					TestConfig.Namespace, TestConfig.RegistryName,
					packageName, version,
				),
			)
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))

			// get package index
			gomega.Eventually(func() bool {
				req = client.NewRequest(
					"GET",
					fmt.Sprintf("/pkg/%s/%s/cargo/%s",
						TestConfig.Namespace, TestConfig.RegistryName,
						getIndexFilePathFromImageName(packageName),
					),
				)
				resp, err = client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
				indexArr, err := getPackageIndexContentFromText(string(resp.Body))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(len(indexArr)).To(gomega.Equal(1))
				gomega.Expect(indexArr[0].Name).To(gomega.Equal(packageName))
				gomega.Expect(indexArr[0].Version).To(gomega.Equal(version))
				return indexArr[0].Yanked
			}, 5, 1).Should(gomega.Equal(false))
		})
	})
}
