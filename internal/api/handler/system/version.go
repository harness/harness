// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package system

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/version"
)

// HandleVersion writes the server version number
// to the http.Response body in plain text.
func HandleVersion(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", version.Version)
}
