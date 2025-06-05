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

var test02Upload = func() {
	ginkgo.Context("Upload", func() {
		ginkgo.BeforeEach(func() {
			if !IsTestEnabled(TestUpload) {
				ginkgo.Skip("Upload tests are disabled")
			}
		})

		ginkgo.It("should upload an artifact", func() {
			// Use a unique artifact name and version for this test
			artifactName := GetUniqueArtifactName("upload", 1)
			version := GetUniqueVersion(1)
			filename := fmt.Sprintf("%s-%s.jar", artifactName, version)
			path := fmt.Sprintf("/maven/%s/%s/com/example/%s/%s/%s",
				TestConfig.Namespace, TestConfig.RegistryName, artifactName, version, filename)

			req := client.NewRequest("PUT", path)
			req.SetHeader("Content-Type", "application/java-archive")
			req.SetBody([]byte("mock jar content"))
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(201))
		})
	})
}
