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

package nuget

import "github.com/harness/gitness/registry/app/pkg/types/nuget"

func buildServiceEndpoint(baseURL string) *nuget.ServiceEndpoint {
	return &nuget.ServiceEndpoint{
		Version: "3.0.0",
		Resources: []nuget.Resource{
			{
				ID:   baseURL + "/query",
				Type: "SearchQueryService",
			},
			{
				ID:   baseURL + "/registration",
				Type: "RegistrationsBaseUrl",
			},
			{
				ID:   baseURL + "/package",
				Type: "PackageBaseAddress/3.0.0",
			},
			{
				ID:   baseURL,
				Type: "PackagePublish/2.0.0",
			},
			{
				ID:   baseURL + "/symbolpackage",
				Type: "SymbolPackagePublish/4.9.0",
			},
			{
				ID:   baseURL + "/query",
				Type: "SearchQueryService/3.0.0-rc",
			},
			{
				ID:   baseURL + "/registration",
				Type: "RegistrationsBaseUrl/3.0.0-rc",
			},
			{
				ID:   baseURL + "/query",
				Type: "SearchQueryService/3.0.0-beta",
			},
			{
				ID:   baseURL + "/registration",
				Type: "RegistrationsBaseUrl/3.0.0-beta",
			},
		},
	}
}
