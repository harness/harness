// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package principal

import (
	"context"

	"github.com/harness/gitness/types"
)

// Controller interface provides an abstraction that allows to have different implementations of
// principal related information.
type Controller interface {
	// List lists the principals based on the provided filter.
	List(ctx context.Context, opts *types.PrincipalFilter) ([]*types.PrincipalInfo, error)
}
