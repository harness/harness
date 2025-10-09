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
			path := fmt.Sprintf("/maven/%s/%s/com/example/test-artifact/1.0/test-artifact-1.0.jar",
				TestConfig.Namespace, TestConfig.RegistryName)
			req := client.NewRequest("PUT", path)
			req.SetHeader("Content-Type", "application/java-archive")
			req.SetBody([]byte("mock jar content"))
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(201))

			// Now download it.
			req = client.NewRequest("GET", path)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("application/java-archive"))
			gomega.Expect(string(resp.Body)).To(gomega.Equal("mock jar content"))
		})
	})
}
