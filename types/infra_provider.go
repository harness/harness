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
	"fmt"
	"strconv"
	"strings"

	"github.com/harness/gitness/types/enum"

	"github.com/docker/go-units"
)

type InfraProviderConfig struct {
	ID         int64                   `json:"-"`
	Identifier string                  `json:"identifier"`
	Name       string                  `json:"name"`
	Type       enum.InfraProviderType  `json:"type"`
	Metadata   map[string]any          `json:"metadata"`
	Resources  []InfraProviderResource `json:"resources"`
	SpaceID    int64                   `json:"-"`
	SpacePath  string                  `json:"space_path"`
	Created    int64                   `json:"created"`
	Updated    int64                   `json:"updated"`
}

type InfraProviderResource struct {
	ID                            int64                  `json:"-"`
	UID                           string                 `json:"identifier"`
	Name                          string                 `json:"name"`
	InfraProviderConfigID         int64                  `json:"-"`
	InfraProviderConfigIdentifier string                 `json:"config_identifier"`
	CPU                           *string                `json:"cpu"`
	Memory                        *string                `json:"memory"`
	Disk                          *string                `json:"disk"`
	Network                       *string                `json:"network"`
	Region                        string                 `json:"region"`
	Metadata                      map[string]string      `json:"metadata"`
	SpaceID                       int64                  `json:"-"`
	SpacePath                     string                 `json:"space_path"`
	InfraProviderType             enum.InfraProviderType `json:"infra_provider_type"`
	Created                       int64                  `json:"created"`
	Updated                       int64                  `json:"updated"`
	IsDeleted                     bool                   `json:"is_deleted,omitempty"`
	Deleted                       *int64                 `json:"deleted,omitempty"`
}

func (i *InfraProviderResource) Identifier() int64 {
	return i.ID
}

func validateInfraProviderResource(a InfraProviderResource) error {
	err := validateCPU(a.CPU)
	if err != nil {
		return err
	}
	err = validateBytes(a.Memory)
	if err != nil {
		return err
	}
	err = validateBytes(a.Disk)
	if err != nil {
		return err
	}
	return nil
}

func validateBytes(bytes *string) error {
	if bytes == nil {
		return fmt.Errorf("bytes is required")
	}
	intValue, err := units.RAMInBytes(withoutSpace(*bytes))
	if err != nil {
		return err
	}
	if intValue < 0 {
		return fmt.Errorf("bytes must be positive")
	}
	return nil
}

func withoutSpace(str string) string {
	return strings.ReplaceAll(str, " ", "")
}

func validateCPU(cpu *string) error {
	if cpu == nil {
		return fmt.Errorf("cpu is required")
	}
	intValue, err := strconv.Atoi(withoutSpace(*cpu))
	if err != nil {
		return err
	}
	if intValue < 0 {
		return fmt.Errorf("cpu must be positive")
	}
	return nil
}

func CompareInfraProviderResource(a, b InfraProviderResource) int {
	// If either is invalid, return 0 since we cant compare them
	err := validateInfraProviderResource(a)
	if err != nil {
		return 0
	}
	err = validateInfraProviderResource(b)
	if err != nil {
		return 0
	}
	cpuA, _ := strconv.Atoi(withoutSpace(*a.CPU))
	cpuB, _ := strconv.Atoi(withoutSpace(*b.CPU))
	if cpuA != cpuB {
		return cpuA - cpuB
	}
	memoryA, _ := units.RAMInBytes(withoutSpace(*a.Memory))
	memoryB, _ := units.RAMInBytes(withoutSpace(*b.Memory))
	if memoryA != memoryB {
		return int(memoryA - memoryB)
	}
	diskA, _ := units.RAMInBytes(withoutSpace(*a.Disk))
	diskB, _ := units.RAMInBytes(withoutSpace(*b.Disk))
	if diskA != diskB {
		return int(diskA - diskB)
	}
	if a.Region != b.Region {
		if a.Region < b.Region {
			return -1
		}
		return 1
	}
	return 0
}

type InfraProviderTemplate struct {
	ID                            int64  `json:"-"`
	Identifier                    string `json:"identifier"`
	InfraProviderConfigID         int64  `json:"-"`
	InfraProviderConfigIdentifier string `json:"config_identifier"`
	Description                   string `json:"description"`
	Data                          string `json:"data"`
	Version                       int64  `json:"-"`
	SpaceID                       int64  `json:"space_id"`
	SpacePath                     string `json:"space_path"`
	Created                       int64  `json:"created"`
	Updated                       int64  `json:"updated"`
}
