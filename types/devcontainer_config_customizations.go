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

	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	GitspaceCustomizationsKey  CustomizationsKey = "harnessGitspaces"
	VSCodeCustomizationsKey    CustomizationsKey = "vscode"
	JetBrainsCustomizationsKey CustomizationsKey = "jetbrains"
)

type CustomizationsKey string

func (ck CustomizationsKey) String() string {
	return string(ck)
}

// DevContainerConfigCustomizations implements various Extract* function to extract out custom field defines in
// customization field in devcontainer.json.
type DevContainerConfigCustomizations map[string]any

// VSCodeCustomizationSpecs contains details about vscode customization.
// eg:
//
//	"customizations": {
//			// Configure properties specific to VS Code.
//			"vscode": {
//				"settings": {
//	       "java.home": "/docker-java-home"
//	     },
//				"extensions": [
//					"streetsidesoftware.code-spell-checker"
//				]
//			}
//		}
type VSCodeCustomizationSpecs struct {
	Extensions []string       `json:"extensions"`
	Settings   map[string]any `json:"settings"`
}

// GitspaceCustomizationSpecs contains details about harness platform connectors and AI agent configuration.
// eg:
//
//	"customizations": {
//	  "harnessGitspaces": {
//	    "connectors": [
//	      {
//	        "type": "DockerRegistry",
//	        "identifier": "testharnessjfrog"
//	      },
//	      {
//	        "type": "Artifactory",
//	        "identifier": "testartifactoryconnector"
//	      }
//	    ],
//	    "ai-agent": {
//	      "type": "claude-code",
//	      "auth": "API-Key",
//	      "secret-ref": "secretref"
//	    }
//	  }
//	}
type GitspaceCustomizationSpecs struct {
	Connectors []struct {
		Type string `json:"type"`
		ID   string `json:"identifier"`
	} `json:"connectors"`
	AIAgent *struct {
		Type      string `json:"type"`
		Auth      string `json:"auth"`
		SecretRef string `json:"secret_ref"`
	} `json:"ai_agent,omitempty"`
}

type JetBrainsBackend string

func (jb JetBrainsBackend) String() string {
	return string(jb)
}

func (jb JetBrainsBackend) Valid() bool {
	_, valid := ValidJetBrainsBackendSet[jb]

	return valid
}

func (jb JetBrainsBackend) IdeType() enum.IDEType {
	var ideType enum.IDEType
	switch jb {
	case IntelliJJetBrainsBackend:
		ideType = enum.IDETypeIntelliJ
	case GolandJetBrainsBackend:
		ideType = enum.IDETypeGoland
	case PyCharmJetBrainsBackend:
		ideType = enum.IDETypePyCharm
	case WebStormJetBrainsBackend:
		ideType = enum.IDETypeWebStorm
	case CLionJetBrainsBackend:
		ideType = enum.IDETypeCLion
	case PhpStormJetBrainsBackend:
		ideType = enum.IDETypePHPStorm
	case RubyMineJetBrainsBackend:
		ideType = enum.IDETypeRubyMine
	case RiderJetBrainsBackend:
		ideType = enum.IDETypeRider
	}

	return ideType
}

const (
	IntelliJJetBrainsBackend JetBrainsBackend = "IntelliJ"
	GolandJetBrainsBackend   JetBrainsBackend = "Goland"
	PyCharmJetBrainsBackend  JetBrainsBackend = "PyCharm"
	WebStormJetBrainsBackend JetBrainsBackend = "WebStorm"
	CLionJetBrainsBackend    JetBrainsBackend = "CLion"
	PhpStormJetBrainsBackend JetBrainsBackend = "PhpStorm"
	RubyMineJetBrainsBackend JetBrainsBackend = "RubyMine"
	RiderJetBrainsBackend    JetBrainsBackend = "Rider"
)

var ValidJetBrainsBackendSet = map[JetBrainsBackend]struct{}{
	IntelliJJetBrainsBackend: {},
	GolandJetBrainsBackend:   {},
	PyCharmJetBrainsBackend:  {},
	WebStormJetBrainsBackend: {},
	CLionJetBrainsBackend:    {},
	PhpStormJetBrainsBackend: {},
	RubyMineJetBrainsBackend: {},
	RiderJetBrainsBackend:    {},
}

type JetBrainsCustomizationSpecs struct {
	Backend JetBrainsBackend `json:"backend"`
	Plugins []string         `json:"plugins"`
}

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

	data, ok := val.(map[string]any)
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

func (dcc DevContainerConfigCustomizations) ExtractJetBrainsSpecs() *JetBrainsCustomizationSpecs {
	data, ok := dcc[JetBrainsCustomizationsKey.String()]
	if !ok {
		// Log that the key is missing, but return nil
		log.Warn().Msgf("JetBrains customization key %q not found, returning empty struct",
			JetBrainsCustomizationsKey)
		return nil
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		// Log the error during marshalling and return nil
		log.Printf("Failed to marshal data for key %q: %v", JetBrainsCustomizationsKey, err)
		return nil
	}

	var jetbrainsSpecs JetBrainsCustomizationSpecs
	if err := json.Unmarshal(rawData, &jetbrainsSpecs); err != nil {
		// Log the error during unmarshalling and return nil
		log.Printf("Failed to unmarshal data for key %q: %v", JetBrainsCustomizationsKey, err)
		return nil
	}

	return &jetbrainsSpecs
}
