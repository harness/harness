// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

func findServiceAccountFromUID(ctx context.Context,
	saStore store.ServiceAccountStore, saUID string) (*types.ServiceAccount, error) {
	return saStore.FindUID(ctx, saUID)
}
