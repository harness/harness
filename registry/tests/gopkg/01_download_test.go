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
	"encoding/json"
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
			packageName := GetUniqueArtifactName("go", 1)
			version := GetUniqueVersion(1)

			// Generate the package payload
			formData := generateGoPackagePayload(packageName, version)

			// Set up the upload request
			req := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/go/upload",
					TestConfig.Namespace, TestConfig.RegistryName,
				),
			)
			req.SetHeader("Content-Type", formData.contentType)
			req.SetBody(formData.body)

			// Send the request and verify the response
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))

			// Now download mod file.
			path := fmt.Sprintf("/pkg/%s/%s/go/%s/@v/%s.mod",
				TestConfig.Namespace, TestConfig.RegistryName, packageName, version,
			)
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(string(resp.Body)).To(gomega.Equal("module " + packageName + "\n"))

			// Now download info file.
			path = fmt.Sprintf("/pkg/%s/%s/go/%s/@v/%s.info",
				TestConfig.Namespace, TestConfig.RegistryName, packageName, version,
			)
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
			var response PackageInfo
			err = json.Unmarshal(resp.Body, &response)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(response.Version).To(gomega.Equal(version))
			gomega.Expect(response.Time).To(gomega.Equal("2023-01-01T00:00:00Z"))

			// Now download zip file.
			path = fmt.Sprintf("/pkg/%s/%s/go/%s/@v/%s.zip",
				TestConfig.Namespace, TestConfig.RegistryName, packageName, version,
			)
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("application/zip"))
			gomega.Expect(string(resp.Body)).To(gomega.Equal("mock package content"))
		})

		ginkgo.It("should download an artifact from upstream", func() {
			packageName := "golang.org/x/time"
			version := "v0.9.0"

			// get package index
			req := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/go/%s/@v/%s.mod",
					TestConfig.Namespace, TestConfig.RegistryName,
					packageName, version,
				),
			)
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
			gomega.Expect(err).To(gomega.BeNil())

			// Now download info file.
			path := fmt.Sprintf("/pkg/%s/%s/go/%s/@v/%s.info",
				TestConfig.Namespace, TestConfig.RegistryName, packageName, version,
			)
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))

			// Now download zip file.
			path = fmt.Sprintf("/pkg/%s/%s/go/%s/@v/%s.zip",
				TestConfig.Namespace, TestConfig.RegistryName, packageName, version,
			)
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("application/zip"))
		})
	})
}
