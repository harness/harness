// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
)

// ParseCheckListOptions extracts the status check list API options from the url.
func ParseCheckListOptions(r *http.Request) types.CheckListOptions {
	return types.CheckListOptions{
		Page: ParsePage(r),
		Size: ParseLimit(r),
	}
}
