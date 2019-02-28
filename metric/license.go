// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package metric

import "github.com/drone/drone/core"

// License registers the license metrics.
func License(license core.LicenseService) {
	// track days until expires
	// track user limit
	// track repo limit
}
