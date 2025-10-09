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

package common

import (
	"context"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

func GenerateOciTokenURL(registryURL string) string {
	return registryURL + "/v2/token"
}

func GenerateSetupClientHostnameAndRegistry(registryURL string) (hostname string, registryRef string) {
	regURL, err := url.Parse(registryURL)
	if err != nil {
		return "", ""
	}
	return regURL.Host, strings.Trim(regURL.Path, "/")
}

func GetHost(ctx context.Context, urlStr string) string {
	if !strings.Contains(urlStr, "://") {
		urlStr = "https://" + urlStr
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Failed to parse URL: %s", urlStr)
		return ""
	}
	return u.Host
}

func GetHostName(ctx context.Context, urlStr string) string {
	if !strings.Contains(urlStr, "://") {
		urlStr = "https://" + urlStr
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Failed to parse URL: %s", urlStr)
		return ""
	}
	return u.Hostname()
}

func TrimURLScheme(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		// Return the original URL if parsing fails
		return urlStr
	}

	// Clear the scheme
	u.Scheme = ""

	// Reconstruct the URL string without the scheme
	return strings.TrimPrefix(u.String(), "//")
}

func ExtractFirstQueryParams(queryParams url.Values) map[string]string {
	queryMap := make(map[string]string)
	for key, values := range queryParams {
		if len(values) > 0 {
			queryMap[key] = values[0]
		}
	}
	return queryMap
}
