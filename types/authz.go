// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "github.com/harness/gitness/types/enum"

type PermissionCheck struct {
	Resource   Resource
	Permission enum.Permission
}

type Resource struct {
	Type       enum.ResourceType
	Identifier string
}
