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

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
)

const (
	GitspaceCustomizationsKey CustomizationsKey = "harnessGitspaces"
	VSCodeCustomizationsKey   CustomizationsKey = "vscode"
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

func (dcc DevContainerConfigCustomizations) ExtractVSCodeSpec() *VSCodeCustomizationSpecs {
	val, ok := dcc[VSCodeCustomizationsKey.String()]
	if !ok {
		// Log that the key is missing, but return nil
		log.Warn().Msgf("VSCode customization key %q not found, returning empty struct",
			VSCodeCustomizationsKey.String())
		return nil
	}

	data, ok := val.(map[string]interface{})
	if !ok {
		// Log the type mismatch and return nil
		log.Warn().Msgf("Unexpected data type for key %q, expected map[string]interface{}, but got %T",
			VSCodeCustomizationsKey.String(), val)
		return nil
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		// Log the error during marshalling and return nil
		log.Printf("Failed to marshal data for key %q: %v", VSCodeCustomizationsKey.String(), err)
		return nil
	}

	var vsCodeCustomizationSpecs VSCodeCustomizationSpecs
	if err := json.Unmarshal(rawData, &vsCodeCustomizationSpecs); err != nil {
		// Log the error during unmarshalling and return nil
		log.Printf("Failed to unmarshal data for key %q: %v", VSCodeCustomizationsKey.String(), err)
		return nil
	}

	return &vsCodeCustomizationSpecs
}

type VSCodeCustomizationSpecs struct {
	Extensions []string               `json:"extensions"`
	Settings   map[string]interface{} `json:"settings"`
}

type GitspaceCustomizationSpecs struct {
	Connectors []struct {
		Type string `json:"type"`
		ID   string `json:"identifier"`
	} `json:"connectors"`
}
