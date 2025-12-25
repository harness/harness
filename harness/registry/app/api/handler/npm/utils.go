//  Copyright 2023 Harness, Inc.
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

package npm

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

func PackageNameFromParams(r *http.Request) string {
	scope, _ := url.PathUnescape(r.PathValue("scope")) // Decode encoded scope
	id, _ := url.PathUnescape(r.PathValue("id"))
	if scope != "" {
		if id != "" {
			return fmt.Sprintf("@%s/%s", scope, id)
		}
		return fmt.Sprintf("@%s", scope)
	}
	return id
}

func GetVersionFromParams(r *http.Request) string {
	version := r.PathValue("version")
	if version == "" {
		var err error
		version, err = VersionNameFromFileName(GetFileName(r))
		if err != nil {
			return ""
		}
		return version
	}
	return version
}

func VersionNameFromFileName(filename string) (string, error) {
	// Define regex pattern: package-name-version.tgz
	re := regexp.MustCompile(`-(\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?)\.tgz$`)

	// Find match
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1], nil
	}
	return "", fmt.Errorf("version not found in filename: %s", filename)
}

func GetFileName(r *http.Request) string {
	scope := r.PathValue("scope")
	file := r.PathValue("filename")
	if scope != "" {
		return fmt.Sprintf("@%s/%s", scope, file)
	}
	return file
}
