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

// ResourceType represents the different types of resources that can be guarded with permissions.
type ResourceType string

const (
	ResourceTypeSpace          ResourceType = "SPACE"
	ResourceTypeRepo           ResourceType = "REPOSITORY"
	ResourceTypeUser           ResourceType = "USER"
	ResourceTypeServiceAccount ResourceType = "SERVICEACCOUNT"
	ResourceTypeService        ResourceType = "SERVICE"
	ResourceTypePipeline       ResourceType = "PIPELINE"
	ResourceTypeSecret         ResourceType = "SECRET"
	ResourceTypeConnector      ResourceType = "CONNECTOR"
	ResourceTypeTemplate       ResourceType = "TEMPLATE"
	ResourceTypeGitspace       ResourceType = "GITSPACE"
	ResourceTypeInfraProvider  ResourceType = "INFRAPROVIDER"
	ResourceTypeRegistry       ResourceType = "REGISTRY"
)

func (ResourceType) Enum() []any {
	return toInterfaceSlice(resourceTypes)
}
func (r ResourceType) Sanitize() (ResourceType, bool) { return Sanitize(r, GetAllResourceTypes) }
func GetAllResourceTypes() ([]ResourceType, ResourceType) {
	return resourceTypes, ""
}

// All valid resource types.
var resourceTypes = sortEnum([]ResourceType{
	ResourceTypeSpace,
	ResourceTypeRepo,
	ResourceTypeUser,
	ResourceTypeServiceAccount,
	ResourceTypeService,
	ResourceTypePipeline,
	ResourceTypeSecret,
	ResourceTypeConnector,
	ResourceTypeTemplate,
	ResourceTypeGitspace,
	ResourceTypeInfraProvider,
	ResourceTypeRegistry,
})

// ParentResourceType defines the different types of parent resources.
type ParentResourceType string

func (ParentResourceType) Enum() []any {
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
