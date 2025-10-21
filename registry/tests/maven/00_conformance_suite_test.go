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
	"testing"

	conformanceutils "github.com/harness/gitness/registry/tests/utils"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var (
	client *conformanceutils.Client
)

func TestMavenConformance(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Maven Registry Conformance Test Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	InitConfig()

	// Log authentication details for debugging
	ginkgo.By("Initializing Maven client with configuration")
	ginkgo.By(fmt.Sprintf("RootURL: %s", TestConfig.RootURL))
	ginkgo.By(fmt.Sprintf("Username: %s", TestConfig.Username))
	ginkgo.By(fmt.Sprintf("Namespace: %s", TestConfig.Namespace))
	ginkgo.By(fmt.Sprintf("RegistryName: %s", TestConfig.RegistryName))
	ginkgo.By(fmt.Sprintf("Password/Token available: %t", TestConfig.Password != ""))

	// Ensure we have a valid token.
	if TestConfig.Password == "" {
		ginkgo.Skip("Skipping integration tests: REGISTRY_PASSWORD environment variable not set")
	}

	// Initialize client with auth token.
	client = conformanceutils.NewClient(TestConfig.RootURL, TestConfig.Password, TestConfig.Debug)
})

var _ = ginkgo.Describe("Maven Registry Conformance Tests", func() {
	// Test categories will be defined in separate files.
	test01Download()
	test02Upload()
	test03ContentDiscovery()
	test05ErrorHandling()
})

// SkipIfDisabled skips the test if the category is disabled.
func SkipIfDisabled(category TestCategory) {
	if !IsTestEnabled(category) {
		ginkgo.Skip(string(category) + " tests are disabled")
	}
}
