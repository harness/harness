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

package enum

// PrincipalType defines the supported types of principals.
type PrincipalType string

func (PrincipalType) Enum() []any                            { return toInterfaceSlice(principalTypes) }
func (s PrincipalType) Sanitize() (PrincipalType, bool)      { return Sanitize(s, GetAllPrincipalTypes) }
func GetAllPrincipalTypes() ([]PrincipalType, PrincipalType) { return principalTypes, "" }

const (
	// PrincipalTypeUser represents a user.
	PrincipalTypeUser PrincipalType = "user"
	// PrincipalTypeServiceAccount represents a service account.
	PrincipalTypeServiceAccount PrincipalType = "serviceaccount"
	// PrincipalTypeService represents a service.
	PrincipalTypeService PrincipalType = "service"
)

var principalTypes = sortEnum([]PrincipalType{
	PrincipalTypeUser,
	PrincipalTypeServiceAccount,
	PrincipalTypeService,
})
