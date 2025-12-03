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

package mavenconformance

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
				artifacts := []struct {
					version  string
					filename string
					fileType string
				}{
					{version: version1, filename: fmt.Sprintf("%s-%s.jar", artifactName, version1), fileType: "jar"},
					{version: version1, filename: fmt.Sprintf("%s-%s.pom", artifactName, version1), fileType: "pom"},
					{version: version2, filename: fmt.Sprintf("%s-%s.jar", artifactName, version2), fileType: "jar"},
					{version: version2, filename: fmt.Sprintf("%s-%s.pom", artifactName, version2), fileType: "pom"},
				}

				for _, artifact := range artifacts {
					artifactPath := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
						TestConfig.Namespace,
						TestConfig.RegistryName,
						artifactName,
						artifact.version,
						artifact.filename,
					)

					contentType := "application/octet-stream"
					if artifact.fileType == "jar" {
						contentType = "application/java-archive"
					} else if artifact.fileType == "pom" {
						contentType = "application/xml"
					}

					req := client.NewRequest("PUT", artifactPath)
					req.SetHeader("Content-Type", contentType)
					req.SetBody([]byte("mock content"))
					resp, err := client.Do(req)
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(resp.StatusCode).To(gomega.Equal(201))
				}
			})

			ginkgo.It("should find artifacts by version", func() {
				// Use the unique artifact name and version defined earlier.
				filename := fmt.Sprintf("%s-%s.jar", artifactName, version1)
				path := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					artifactName,
					version1,
					filename,
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(string(resp.Body)).To(gomega.Equal("mock content"))
			})

			ginkgo.It("should find POM files", func() {
				// Use the unique artifact name and version defined earlier.
				filename := fmt.Sprintf("%s-%s.pom", artifactName, version1)
				path := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					artifactName,
					version1,
					filename,
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
				gomega.Expect(string(resp.Body)).To(gomega.Equal("mock content"))
			})

			ginkgo.It("should handle non-existent artifacts", func() {
				// Use a completely different artifact name and version for non-existent artifact.
				nonExistentArtifact := GetUniqueArtifactName("nonexistent", 99)
				nonExistentVersion := GetUniqueVersion(99)
				filename := fmt.Sprintf("%s-%s.jar", nonExistentArtifact, nonExistentVersion)
				path := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					nonExistentArtifact,
					nonExistentVersion,
					filename,
				)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
			})
		})
	})
}
