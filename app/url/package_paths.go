// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package url

import (
	"context"
	"fmt"
	"path"
)

// PackagePathSpec marks structs as package URL path specifications.
type PackagePathSpec interface {
	packagePathSpec()
}

// PythonFilePathSpec specifies the path for a Python package file download.
type PythonFilePathSpec struct {
	RootIdentifier string
	RegIdentifier  string
	Image          string
	Version        string
	Filename       string
}

func (PythonFilePathSpec) packagePathSpec() {}

// PackagePathFor returns the URL path for a given package path spec.
func (p *provider) PackagePathFor(_ context.Context, spec PackagePathSpec) (string, error) {
	switch v := spec.(type) {
	case PythonFilePathSpec:
		return "../../" + path.Join("_", v.RegIdentifier, v.Image, v.Version, v.Filename), nil
	default:
		return "", fmt.Errorf("unknown package path spec: %#v", v)
	}
}
