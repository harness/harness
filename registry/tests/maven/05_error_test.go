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
	"log"
	"time"

	conformanceutils "github.com/harness/gitness/registry/tests/utils"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test05ErrorHandling = func() {
	ginkgo.Context("Error Handling", func() {
		ginkgo.Context("Invalid Requests", func() {
			ginkgo.It("should reject invalid artifact path", func() {
				path := fmt.Sprintf("/maven/%s/%s/invalid/path",
					TestConfig.Namespace,
					TestConfig.RegistryName)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				log.Printf("Response for invalid path: %d %s\n",
					resp.StatusCode, string(resp.Body))
				log.Printf("Response Headers: %v\n", resp.Headers)
				// The server returns 500 with a text/plain error response.
				gomega.Expect(resp.StatusCode).To(gomega.Equal(500))
				gomega.Expect(resp.Headers.Get("Content-Type")).To(
					gomega.Equal("application/json; charset=utf-8"))
				gomega.Expect(string(resp.Body)).To(gomega.ContainSubstring("invalid path format"))
			})

			ginkgo.It("should reject invalid version format", func() {
				path := fmt.Sprintf("/maven/%s/%s/com/example/test-artifact/invalid-version/test-artifact.jar",
					TestConfig.Namespace,
					TestConfig.RegistryName)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
			})

			ginkgo.It("should reject invalid groupId", func() {
				path := fmt.Sprintf("/maven/%s/%s/com/example/../test-artifact/1.0/test-artifact-1.0.jar",
					TestConfig.Namespace,
					TestConfig.RegistryName)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
			})
		})

		ginkgo.Context("Authentication", func() {
			ginkgo.It("should reject unauthorized access", func() {
				// Create a new client with invalid credentials.
				invalidClient := conformanceutils.NewClient(TestConfig.RootURL, "invalid", TestConfig.Debug)
				path := fmt.Sprintf("/maven/%s/%s/com/example/test-artifact/1.0/test-artifact-1.0.jar",
					TestConfig.Namespace,
					TestConfig.RegistryName)
				req := invalidClient.NewRequest("GET", path)
				resp, err := invalidClient.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(401))
			})

			ginkgo.It("should reject access to non-existent space", func() {
				path := fmt.Sprintf("/maven/nonexistent/%s/com/example/test-artifact/1.0/test-artifact-1.0.jar",
					TestConfig.RegistryName)
				log.Printf("Testing path: %s\n", path)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				// The server returns 500 with a text/plain error response.
				gomega.Expect(resp.StatusCode).To(gomega.Equal(500))
				gomega.Expect(resp.Headers.Get("Content-Type")).To(
					gomega.Equal("application/json; charset=utf-8"))
				gomega.Expect(string(resp.Body)).To(gomega.ContainSubstring("ROOT_NOT_FOUND"))
			})
		})

		ginkgo.Context("Content Validation", ginkgo.Ordered, func() {
			// Define variables at the context level so they're accessible to all tests.
			var errorArtifact string
			var errorVersion string

			// Use BeforeAll to set up artifacts just once for all tests in this context.
			ginkgo.BeforeAll(func() {
				// Define a unique artifact name and version for error handling tests with random component.
				errorArtifact = GetUniqueArtifactName("error", int(time.Now().UnixNano()%1000))
				errorVersion = GetUniqueVersion(50)

				// Create test artifact directory with unique name and version.
				filename := fmt.Sprintf("%s-%s.jar", errorArtifact, errorVersion)
				path := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					errorArtifact,
					errorVersion,
					filename)
				req := client.NewRequest("PUT", path)
				req.SetHeader("Content-Type", "application/java-archive")
				req.SetBody([]byte("mock jar content"))
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(201))
			})

			ginkgo.It("should handle invalid POM XML", func() {
				// Use a completely different artifact name with timestamp to ensure uniqueness.
				timestamp := time.Now().UnixNano()
				pomArtifact := fmt.Sprintf("pom-artifact-%d", timestamp)
				pomVersion := fmt.Sprintf("3.0.%d", timestamp)
				filename := fmt.Sprintf("%s-%s.pom", pomArtifact, pomVersion)
				path := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					pomArtifact,
					pomVersion,
					filename)
				req := client.NewRequest("PUT", path)
				req.SetHeader("Content-Type", "text/xml")
				req.SetBody([]byte("invalid xml content"))
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				// The server accepts invalid XML as it's just stored as bytes.
				gomega.Expect(resp.StatusCode).To(gomega.Equal(201))
			})

			ginkgo.It("should handle mismatched content type", func() {
				// Use a completely different artifact name with timestamp to ensure uniqueness.
				timestamp := time.Now().UnixNano()
				mismatchArtifact := fmt.Sprintf("mismatch-artifact-%d", timestamp)
				mismatchVersion := fmt.Sprintf("2.0.%d", timestamp)
				filename := fmt.Sprintf("%s-%s.jar", mismatchArtifact, mismatchVersion)
				path := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
					TestConfig.Namespace,
					TestConfig.RegistryName,
					mismatchArtifact,
					mismatchVersion,
					filename)
				req := client.NewRequest("PUT", path)
				req.SetHeader("Content-Type", "text/plain")
				req.SetBody([]byte("invalid jar content"))
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				// The server accepts any content type as it's just stored as bytes.
				gomega.Expect(resp.StatusCode).To(gomega.Equal(201))
			})
		})
	})
}
