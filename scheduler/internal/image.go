// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package internal

import (
	"fmt"

	"github.com/drone/drone/version"
)

var defaultImage = fmt.Sprintf(
	"drone/controller:%s",
	version.Version.String(),
)

// DefaultImage returns the default dispatch image if none
// is specified.
func DefaultImage(image string) string {
	if image == "" {
		return defaultImage
	}
	return image
}
