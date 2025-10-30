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

package gopkgconformance

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test03ContentDiscovery = func() {
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
				artifacts := []*UploadFormData{
					generateGoPackagePayload(artifactName, version1),
					generateGoPackagePayload(artifactName, version2),
				}

				for _, artifact := range artifacts {
					artifactPath := fmt.Sprintf("/pkg/%s/%s/go/upload",
						TestConfig.Namespace, TestConfig.RegistryName,
					)
					req := client.NewRequest("PUT", artifactPath)
					req.SetHeader("Content-Type", artifact.contentType)
					req.SetBody(artifact.body)
					resp, err := client.Do(req)
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				}
				time.Sleep(2 * time.Second)
			})

			ginkgo.It("should find all versions in package index", func() {
				// Use the unique artifact name and version defined earlier.
				path := fmt.Sprintf("/pkg/%s/%s/go/%s/@v/list",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					artifactName,
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(string(resp.Body)).Should(gomega.ContainSubstring(version1))
				gomega.Expect(string(resp.Body)).Should(gomega.ContainSubstring(version2))
			})

			ginkgo.It("should handle non-existent artifacts", func() {
				// Use a completely different artifact name and version for non-existent artifact.
				nonExistentArtifact := GetUniqueArtifactName("nonexistent", 99)
				nonExistentVersion := GetUniqueVersion(99)
				// verify package index
				path := fmt.Sprintf("/pkg/%s/%s/go/%s/@v/%s.mod",
					TestConfig.Namespace, TestConfig.RegistryName,
					nonExistentArtifact, nonExistentVersion,
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
			})
		})
	})
}
