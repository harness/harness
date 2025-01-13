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

package utils

import (
	"regexp"
	"strings"
)

// ConvertOptionsToEnvVariables converts the option keys to standardised env variables using the
// devcontainer specification to ensure uniformity in the naming and casing of the env variables.
// Reference: https://containers.dev/implementors/features/#option-resolution
func ConvertOptionsToEnvVariables(str string) string {
	// Replace all non-alphanumeric characters (excluding underscores) with '_'
	reNonAlnum := regexp.MustCompile(`[^\w_]`)
	str = reNonAlnum.ReplaceAllString(str, "_")

	// Replace leading digits or underscores with a single '_'
	reLeadingDigitsOrUnderscores := regexp.MustCompile(`^[\d_]+`)
	str = reLeadingDigitsOrUnderscores.ReplaceAllString(str, "_")

	// Convert the string to uppercase
	str = strings.ToUpper(str)

	return str
}
