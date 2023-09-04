// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// ParentResourceType defines the different types of parent resources.
type ParentResourceType string

func (ParentResourceType) Enum() []interface{} {
	return toInterfaceSlice(GetAllParentResourceTypes())
}

var (
	ParentResourceTypeSpace ParentResourceType = "space"
	ParentResourceTypeRepo  ParentResourceType = "repo"
)

func GetAllParentResourceTypes() []ParentResourceType {
	return []ParentResourceType{
		ParentResourceTypeSpace,
		ParentResourceTypeRepo,
	}
}