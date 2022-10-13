// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"github.com/harness/gitness/types"
)

// Service returns true if the Service if valid.
func Service(sa *types.Service) error {
	// validate UID
	if err := UID(sa.UID); err != nil {
		return err
	}

	// validate name
	if err := Name(sa.Name); err != nil {
		return err
	}

	return nil
}
