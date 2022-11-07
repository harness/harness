// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"github.com/harness/gitness/types"
)

// Service returns true if the Service is valid.
type Service func(*types.Service) error

// ServiceDefault is the default Service validation.
func ServiceDefault(svc *types.Service) error {
	// validate UID
	if err := UID(svc.UID); err != nil {
		return err
	}

	// Validate Email
	if err := Email(svc.Email); err != nil {
		return err
	}

	// validate DisplayName
	if err := DisplayName(svc.DisplayName); err != nil {
		return err
	}

	return nil
}
