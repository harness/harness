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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test04ContentDiscovery = func() {
	ginkgo.Context("Content Discovery", func() {
		ginkgo.BeforeEach(func() {
			if !IsTestEnabled(TestContentDiscovery) {
				ginkgo.Skip("Content discovery tests are disabled")
			}
		})

		ginkgo.Context("Basic Content Discovery", ginkgo.Ordered, func() {
			// Define variables at the context level so they're accessible to all tests.
			var artifactName string
			var version1, version2 string

			// Use BeforeAll to set up artifacts just once for all tests in this context.
			ginkgo.BeforeAll(func() {
				// Define unique artifact name for this test context with random component
				artifactName = GetUniqueArtifactName("discovery", int(time.Now().UnixNano()%1000))
				// Define test versions that will be used across all tests in this context.
				version1 = GetUniqueVersion(10)
				version2 = GetUniqueVersion(11)
				// Upload test artifacts with unique names and versions.
				artifacts := [][]byte{
					generateCargoPackagePayload(artifactName, version1),
					generateCargoPackagePayload(artifactName, version2),
				}

				for _, artifact := range artifacts {
					artifactPath := fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/new",
						TestConfig.Namespace, TestConfig.RegistryName,
					)
					contentType := "application/crate"
					req := client.NewRequest("PUT", artifactPath)
					req.SetHeader("Content-Type", contentType)
					req.SetBody(artifact)
					resp, err := client.Do(req)
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(resp.StatusCode).To(gomega.Equal(201))
				}
			})

			ginkgo.It("should find artifacts by version", func() {
				// Use the unique artifact name and version defined earlier.
				path := fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/%s/%s/download",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					artifactName,
					version1,
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(string(resp.Body)).To(gomega.Equal("mock package content"))
			})

			ginkgo.It("should find all versions in package index", func() {
				// Use the unique artifact name and version defined earlier.
				time.Sleep(2 * time.Second)
				var indexArr []packageIndexMetadata
				path := fmt.Sprintf("/pkg/%s/%s/cargo/%s",
					TestConfig.Namespace, TestConfig.RegistryName,
					getIndexFilePathFromImageName(artifactName),
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
				indexArr, err = getPackageIndexContentFromText(string(resp.Body))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(len(indexArr)).To(gomega.Equal(2))
				gomega.Expect(indexArr[0].Name).To(gomega.Equal(artifactName))
				gomega.Expect(indexArr[0].Version).To(gomega.Equal(version1))
				gomega.Expect(indexArr[0].Yanked).To(gomega.Equal(false))
				gomega.Expect(indexArr[1].Name).To(gomega.Equal(artifactName))
				gomega.Expect(indexArr[1].Version).To(gomega.Equal(version2))
				gomega.Expect(indexArr[1].Yanked).To(gomega.Equal(false))
			})

			ginkgo.It("should handle non-existent artifacts", func() {
				// Use a completely different artifact name and version for non-existent artifact.
				nonExistentArtifact := GetUniqueArtifactName("nonexistent", 99)
				nonExistentVersion := GetUniqueVersion(99)
				// verify package index
				path := fmt.Sprintf("/pkg/%s/%s/cargo/%s",
					TestConfig.Namespace, TestConfig.RegistryName,
					getIndexFilePathFromImageName(nonExistentArtifact),
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))

				// verify download package file
				path = fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/%s/%s/download",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					nonExistentArtifact,
					nonExistentVersion,
				)
				req = client.NewRequest("GET", path)
				resp, err = client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))

			})
		})
	})
}
