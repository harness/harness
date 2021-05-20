// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

import "github.com/drone/drone/core"

var (
	dummyTemplate = &core.Template{
		Name:    "my_template",
		Data:    "my_data",
		Created: 1,
		Updated: 2,
	}
	dummyTemplateList = []*core.Template{
		dummyTemplate,
	}
)
