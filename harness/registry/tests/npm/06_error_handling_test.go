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

package npmconformance

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test06ErrorHandling = func() {
	ginkgo.Context("Error Handling", func() {
		ginkgo.BeforeEach(func() {
			SkipIfDisabled(TestErrorHandling)
		})

		ginkgo.It("should return 404 for non-existent package metadata", func() {
			nonExistentPackage := "non-existent-package-12345"

			req := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, nonExistentPackage,
				),
			)

			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
		})

		ginkgo.It("should return 404 for non-existent package file download", func() {
			nonExistentPackage := "non-existent-package-54321"
			version := "1.0.0"
			filename := fmt.Sprintf("%s-%s.tgz", nonExistentPackage, version)

			req := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/-/%s/%s",
					TestConfig.Namespace, TestConfig.RegistryName, nonExistentPackage, version, filename,
				),
			)

			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
		})

		ginkgo.It("should return 404 for non-existent scoped package", func() {
			scope := "nonexistent"
			packageName := "test-package"

			req := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)

			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(404))
		})

		ginkgo.It("should return empty object for non-existent package tags", func() {
			nonExistentPackage := "non-existent-tags-package"

			req := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/",
					TestConfig.Namespace, TestConfig.RegistryName, nonExistentPackage,
				),
			)

			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))

			// Verify response body is empty object
			body := resp.Body
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(string(body)).To(gomega.Equal("{}"))

		})

		ginkgo.It("should return 404 when trying to delete non-existent tag", func() {
			packageName := GetUniquePackageName("delete-nonexistent-tag", 2)
			version := GetUniqueVersion(2)
			nonExistentTag := "nonexistent-tag"

			// First upload a package
			packageData := generateNpmPackagePayload(packageName, version, false, "")
			uploadReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)
			uploadReq.SetHeader("Content-Type", "application/json")
			uploadReq.SetBody(packageData)

			uploadResp, err := client.Do(uploadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(uploadResp.StatusCode).To(gomega.Equal(200))

			// Try to delete non-existent tag
			deleteTagReq := client.NewRequest(
				"DELETE",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/%s",
					TestConfig.Namespace, TestConfig.RegistryName, packageName, nonExistentTag,
				),
			)

			_, err = client.Do(deleteTagReq)
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.It("should handle invalid version format in tag operations", func() {
			packageName := GetUniquePackageName("invalid-version", 3)
			version := GetUniqueVersion(3)
			invalidVersion := "not-a-valid-version"

			// First upload a package
			packageData := generateNpmPackagePayload(packageName, version, false, "")
			uploadReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)
			uploadReq.SetHeader("Content-Type", "application/json")
			uploadReq.SetBody(packageData)

			uploadResp, err := client.Do(uploadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(uploadResp.StatusCode).To(gomega.Equal(200))

			// Try to add tag with invalid version
			addTagReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/beta",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)
			addTagReq.SetHeader("Content-Type", "application/json")
			addTagReq.SetBody(fmt.Sprintf(`"%s"`, invalidVersion))

			addTagResp, err := client.Do(addTagReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(addTagResp.StatusCode).To(gomega.Equal(404)) // Bad Request, Not Found, or Unprocessable Entity
		})

		ginkgo.It("should return 404 for HEAD request on non-existent file", func() {
			nonExistentPackage := "non-existent-head-test"
			filename := "nonexistent-file.tgz"

			req := client.NewRequest(
				"HEAD",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/-/%s",
					TestConfig.Namespace, TestConfig.RegistryName, nonExistentPackage, filename,
				),
			)

			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(string(resp.Body)).To(gomega.Equal(""))
		})
	})
}
