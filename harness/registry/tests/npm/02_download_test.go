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
	"encoding/base64"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test02Download = func() {
	ginkgo.Context("Download", func() {
		ginkgo.BeforeEach(func() {
			SkipIfDisabled(TestDownload)
		})

		ginkgo.It("should download a non-scoped NPM package file", func() {
			packageName := GetUniquePackageName("download", 1)
			version := GetUniqueVersion(1)
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)

			// Generate expected tarball content for validation
			expectedTarballContent := fmt.Sprintf(`{"name":"%s","version":"%s","main":"index.js"}`, packageName, version)

			// First upload the package
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

			// Now download the package file
			downloadReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/-/%s/%s",
					TestConfig.Namespace, TestConfig.RegistryName, packageName, version, filename,
				),
			)

			downloadResp, err := client.Do(downloadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(downloadResp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(downloadResp.Headers.Get("Content-Disposition")).To(
				gomega.ContainSubstring(fmt.Sprintf("filename=%s", filename)))

			// Validate file content
			gomega.Expect(downloadResp.Body).ToNot(gomega.BeEmpty(), "Downloaded file should not be empty")

			// The downloaded content should be the base64-decoded tarball content
			// Since we're mocking the tarball as JSON content, we can validate it directly
			downloadedContent := string(downloadResp.Body)

			// Try to decode as base64 first (in case the server returns base64-encoded content)
			if decodedContent, err := base64.StdEncoding.DecodeString(downloadedContent); err == nil {
				downloadedContent = string(decodedContent)
			}

			// Validate that the downloaded content matches what we uploaded
			gomega.Expect(downloadedContent).To(gomega.Equal(expectedTarballContent),
				"Downloaded file content should match the uploaded tarball content")

			// Validate Content-Disposition header contains the correct filename
			gomega.Expect(downloadResp.Headers.Get("Content-Disposition")).To(
				gomega.ContainSubstring(fmt.Sprintf("filename=%s", filename)))

		})

		ginkgo.It("should download a scoped NPM package file", func() {
			scope := TestScope
			packageName := "test-scoped-download"
			version := GetUniqueVersion(2)
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)

			// First upload the scoped package
			packageData := generateNpmPackagePayload(packageName, version, true, scope)
			uploadReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)
			uploadReq.SetHeader("Content-Type", "application/json")
			uploadReq.SetBody(packageData)

			uploadResp, err := client.Do(uploadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(uploadResp.StatusCode).To(gomega.Equal(200))

			// Now download the scoped package file
			downloadReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/-/%s/@%s/%s",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName, version, scope, filename,
				),
			)

			downloadResp, err := client.Do(downloadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(downloadResp.StatusCode).To(gomega.Equal(200))
			expectedFilename := fmt.Sprintf("filename=%s", "@"+scope+"/"+filename)
			gomega.Expect(downloadResp.Headers.Get("Content-Disposition")).To(
				gomega.ContainSubstring(expectedFilename))
		})

		ginkgo.It("should download package file by name", func() {
			packageName := GetUniquePackageName("download-by-name", 3)
			version := GetUniqueVersion(3)
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)

			// First upload the package
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

			// Now download the package file by name
			downloadReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/-/%s",
					TestConfig.Namespace, TestConfig.RegistryName, packageName, filename,
				),
			)

			downloadResp, err := client.Do(downloadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(downloadResp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(downloadResp.Headers.Get("Content-Disposition")).To(
				gomega.ContainSubstring(fmt.Sprintf("filename=%s", filename)))
		})

		ginkgo.It("should perform HEAD request on package file", func() {
			packageName := GetUniquePackageName("head-test", 4)
			version := GetUniqueVersion(4)
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)

			// First upload the package
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

			// Now perform HEAD request
			headReq := client.NewRequest(
				"HEAD",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/-/%s",
					TestConfig.Namespace, TestConfig.RegistryName, packageName, filename,
				),
			)

			headResp, err := client.Do(headReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(headResp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(len(headResp.Body)).To(gomega.Equal(0)) // HEAD should not return body
		})
	})
}
