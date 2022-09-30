// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package common

import "github.com/harness/gitness/types/enum"

// CreatePathRequest used for path creation apis.
type CreatePathRequest struct {
	Path string `json:"path"`
}

// CreateServiceAccountRequest used for service account creation apis.
type CreateServiceAccountRequest struct {
	UID        string                  `json:"uid"`
	Name       string                  `json:"name"`
	ParentType enum.ParentResourceType `json:"parentType"`
	ParentID   int64                   `json:"parentId"`
}
