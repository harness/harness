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

package types

import "encoding/json"

const (
	GitspaceCustomizationsKey CustomizationsKey = "harnessGitspaces"
)

type CustomizationsKey string

func (ck CustomizationsKey) String() string {
	return string(ck)
}

type DevContainerConfigCustomizations map[string]interface{}

func (dcc DevContainerConfigCustomizations) ExtractGitspaceSpec() *GitspaceCustomizationSpecs {
	val, ok := dcc[GitspaceCustomizationsKey.String()]
	if !ok {
		return nil
	}

	// val has underlying map[string]interface{} type as it is default for JSON objects
	// converting to json so that val can be marshaled to GitspaceCustomizationSpecs type.
	rawData, _ := json.Marshal(&val)

	var gitspaceSpecs GitspaceCustomizationSpecs
	if err := json.Unmarshal(rawData, &gitspaceSpecs); err != nil {
		return nil
	}
	return &gitspaceSpecs
}

type GitspaceCustomizationSpecs struct {
	Connectors []struct {
		Type string `json:"type"`
		ID   string `json:"identifier"`
	} `json:"connectors"`
}
