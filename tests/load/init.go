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

package load

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("../../.local.env")
	if err != nil {
		log.Println("Error loading .local.env file")
	}
	err = godotenv.Load("../../.test.env")
	if err != nil {
		log.Println("Error loading .test.env file")
	}
}

func isE2ETestsEnabled() bool {
	env := os.Getenv("GITNESS_E2E_TEST_ENABLED")
	env = strings.ToLower(env)
	env = strings.TrimSpace(env)
	return env == "1" || env == "true"
}
