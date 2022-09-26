// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// ParentResourceType defines the different types of parent resources.
type ParentResourceType string

var (
	ParentResourceTypeSpace ParentResourceType = "space"
	ParentResourceTypeRepo  ParentResourceType = "repo"
)
