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
	"log"

	"github.com/onsi/ginkgo/v2"
)

func init() {
	// Register a reporter hook with Ginkgo.
	ginkgo.ReportAfterSuite("Maven Conformance Report", func(_ ginkgo.Report) {
		// Save the report when the suite is done.
		// Pass false to avoid duplicate logging from the shell script.
		if err := SaveReport("maven_conformance_report.json", false); err != nil {
			// Log error but don't fail the test.
			log.Printf("Failed to save report: %v", err)
		}
	})
}
