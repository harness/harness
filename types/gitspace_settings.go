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
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/harness/gitness/types/enum"
)

type GitspaceSettingsFilter struct {
	ListQueryFilter
}

type CriteriaKey string
type SettingsData struct {
	Data     map[string]any           `json:"data,omitempty"`     // generic, user-defined
	Criteria GitspaceSettingsCriteria `json:"criteria,omitempty"` // criteria for the settings
}

type GitspaceSettings struct {
	ID           int64                     `json:"-"`
	Settings     SettingsData              `json:"settings"`
	SettingsType enum.GitspaceSettingsType `json:"settings_type,omitempty"`
	CriteriaKey  CriteriaKey               `json:"criteria_key,omitempty"`
	SpaceID      int64                     `json:"-"`
	Created      int64                     `json:"created,omitempty"`
	Updated      int64                     `json:"updated,omitempty"`
}
type GitspaceSettingsCriteria map[string]any

var ApplyAlwaysToSpaceCriteria = GitspaceSettingsCriteria{}

func flattenCriteria(prefix string, input map[string]any, out map[string]string) {
	const maxDepth = 10 // Add depth protection
	flattenCriteriaWithDepth(prefix, input, out, 0, maxDepth)
}

func flattenCriteriaWithDepth(prefix string, input map[string]any, out map[string]string, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}
	for k, v := range input {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch child := v.(type) {
		case map[string]any:
			flattenCriteriaWithDepth(key, child, out, depth+1, maxDepth)
		default:
			out[key] = fmt.Sprintf("%v", v)
		}
	}
}

func (c GitspaceSettingsCriteria) ToKey() (CriteriaKey, error) {
	if len(c) == 0 {
		return "", nil
	}

	flat := make(map[string]string)
	flattenCriteria("", c, flat)

	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		parts = append(parts, k, flat[k])
	}

	return CriteriaKey(strings.Join(parts, "/")), nil
}

type IDESettings struct {
	AccessList *AccessList[enum.IDEType] `json:"access_list,omitempty"` // Access control for IDEs
	DisableSSH bool                      `json:"disable_ssh"`
}

type AccessList[T comparable] struct {
	Mode ListMode `json:"mode"`
	List []T      `json:"list"`
}

func (a *AccessList[T]) IsAllowed(item T) bool {
	if a == nil || len(a.List) == 0 {
		return true // If no list, assume unrestricted
	}
	found := slices.Contains(a.List, item)
	if a.Mode == ListModeAllow {
		return found
	}
	// ListModeDeny
	return !found
}

func (a *AccessList[T]) Remove(item T) bool {
	if a == nil || len(a.List) == 0 {
		return false
	}
	for i, entry := range a.List {
		if entry == item {
			a.List = append(a.List[:i], a.List[i+1:]...)
			return true
		}
	}
	return false
}

type ListMode string

const (
	ListModeAllow ListMode = "allow"
	ListModeDeny  ListMode = "deny"
)

type SCMProviderSettings struct {
	AccessList *AccessList[enum.GitspaceCodeRepoType] `json:"access_list,omitempty"` // Allow/Deny list for SCM providers
}

type DevcontainerSettings struct {
	DevcontainerImage DevcontainerImage `json:"devcontainer_image"` // Devcontainer image settings
}

type DevcontainerImage struct {
	AccessList        *AccessList[string] `json:"access_list,omitempty"`         // Allow/Deny list for container images
	ImageConnectorRef string              `json:"image_connector_ref,omitempty"` // Connector reference for the image
	ImageName         string              `json:"image_name,omitempty"`          // Name of the container image
}

type GitspaceConfigSettings struct {
	IDEs         IDESettings          `json:"ide"`          // allow list of IDEs
	SCMProviders SCMProviderSettings  `json:"scm"`          // allow list of SCMs
	Devcontainer DevcontainerSettings `json:"devcontainer"` // allow list of devcontainer images
}
type InfraProviderSettings struct {
	AccessList             *AccessList[string]    `json:"access_list,omitempty"`
	AutoStoppingTimeInMins *int                   `json:"auto_stopping_time_in_mins,omitempty"`
	InfraProviderType      enum.InfraProviderType `json:"infra_provider_type,omitempty"`
}

var settingsTypeRegistry = map[enum.GitspaceSettingsType]reflect.Type{
	enum.SettingsTypeInfraProvider:  reflect.TypeOf(InfraProviderSettings{}),
	enum.SettingsTypeGitspaceConfig: reflect.TypeOf(GitspaceConfigSettings{}),
}

func DecodeSettings[T any](data map[string]any) (*T, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(jsonData, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func ValidateAndDecodeSettings(settingsType enum.GitspaceSettingsType, data map[string]any) (any, error) {
	typ, ok := settingsTypeRegistry[settingsType]
	if !ok {
		return nil, fmt.Errorf("no schema registered for settings type: %s", settingsType)
	}

	// Allocate a new struct
	ptr := reflect.New(typ).Interface()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jsonData, ptr); err != nil {
		return nil, fmt.Errorf("schema mismatch: %w", err)
	}
	return ptr, nil
}
