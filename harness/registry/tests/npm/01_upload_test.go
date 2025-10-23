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

var test01Upload = func() {
	ginkgo.Context("Upload", func() {
		ginkgo.BeforeEach(func() {
			SkipIfDisabled(TestUpload)
		})

		ginkgo.It("should upload a non-scoped NPM package", func() {
			packageName := GetUniquePackageName("upload", 1)
			version := GetUniqueVersion(1)

			// Generate the package payload
			packageData := generateNpmPackagePayload(packageName, version, false, "")

			// Set up the upload request
			req := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)
			req.SetHeader("Content-Type", "application/json")
			req.SetBody(packageData)

			// Send the request and verify the response
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
		})

		ginkgo.It("should upload a scoped NPM package", func() {
			scope := TestScope
			packageName := "test-scoped-package"
			version := GetUniqueVersion(2)

			// Generate the scoped package payload
			packageData := generateNpmPackagePayload(packageName, version, true, scope)

			// Set up the upload request for scoped package
			req := client.NewRequest(
				"PUT",
				fmt.Sprintf("/npm/%s/%s/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)
			req.SetHeader("Content-Type", "application/json")
			req.SetBody(packageData)

			// Send the request and verify the response
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
		})
	})
}
