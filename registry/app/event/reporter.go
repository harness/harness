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

package event

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
)

type PackageType int32

type ArtifactDetails struct {
	RegistryID   int64       `json:"registry_id,omitempty"`
	RegistryName string      `json:"registry_name,omitempty"`
	ImagePath    string      `json:"image_path,omitempty"` // format = image:tag
	PackageType  PackageType `json:"package_type,omitempty"`
	ManifestID   int64       `json:"manifest_id,omitempty"`
}

// PackageType constants using iota.
const (
	PackageTypeDOCKER = iota
	PackageTypeGENERIC
	PackageTypeHELM
	PackageTypeMAVEN
)

var PackageTypeValue = map[string]PackageType{
	string(artifact.PackageTypeDOCKER):  PackageTypeDOCKER,
	string(artifact.PackageTypeGENERIC): PackageTypeGENERIC,
	string(artifact.PackageTypeHELM):    PackageTypeHELM,
	string(artifact.PackageTypeMAVEN):   PackageTypeMAVEN,
}

// GetPackageTypeFromString returns the PackageType constant corresponding to the given string value.
func GetPackageTypeFromString(value string) (PackageType, error) {
	if val, ok := PackageTypeValue[value]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("invalid PackageType string value: %s", value)
}

type Reporter interface {
	ReportEvent(
		ctx context.Context, payload any, spacePath string,
	) // format of spacePath = acctID/orgID/projectID
}

type Noop struct {
}

func (*Noop) ReportEvent(_ context.Context, _ any, _ string) {
	// no implementation
}
