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
	"log"

	conformanceutils "github.com/harness/gitness/registry/tests/utils"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test05ErrorHandling = func() {
	ginkgo.Context("Error Handling", func() {
		ginkgo.Context("Invalid Requests", func() {
			ginkgo.It("should reject invalid artifact path", func() {
				path := fmt.Sprintf("/pkg/%s/%s/cargo/invalid/path",
					TestConfig.Namespace,
					TestConfig.RegistryName)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				log.Printf("Response for invalid path: %d %s\n",
					resp.StatusCode, string(resp.Body))
				log.Printf("Response Headers: %v\n", resp.Headers)
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
			})
		})

		ginkgo.Context("Authentication", func() {
			ginkgo.It("should reject unauthorized access", func() {
				// Create a new client with invalid credentials.
				invalidClient := conformanceutils.NewClient(TestConfig.RootURL, "invalid", TestConfig.Debug)
				path := fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/test/1.0.0/download",
					TestConfig.Namespace,
					TestConfig.RegistryName)
				req := invalidClient.NewRequest("GET", path)
				resp, err := invalidClient.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(401))
			})

			ginkgo.It("should reject access to non-existent space", func() {
				path := fmt.Sprintf("/pkg/nonexistent/%s/cargo/api/v1/crates/test/1.0.0/download",
					TestConfig.RegistryName)
				log.Printf("Testing path: %s\n", path)
				req := client.NewRequest("GET", path)
				resp, err := client.Do(req)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
				gomega.Expect(resp.Headers.Get("Content-Type")).To(
					gomega.Equal("application/json; charset=utf-8"))
				gomega.Expect(string(resp.Body)).To(gomega.ContainSubstring("Root not found: nonexistent"))
			})
		})

	})
}
