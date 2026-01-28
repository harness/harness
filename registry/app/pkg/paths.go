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

package pkg

import (
	"fmt"
	"path"
)

const packagePathRoot = "/"

// PackagePathSpec is a type to mark structs as path specs for packages
// Keeping it public so that anyone can override this.
type PackagePathSpec interface {
	pathSpec()
}

type huggingFaceTypeSpec struct {
	imageType string
	imageName string
	path      string
}

func (h huggingFaceTypeSpec) pathSpec() {}

func PathFor(spec PackagePathSpec) (string, error) {
	rootPrefix := []string{packagePathRoot}
	switch v := spec.(type) {
	case huggingFaceTypeSpec:
		blobsPathPrefix := rootPrefix
		blobsPathPrefix = append(blobsPathPrefix, v.imageType, v.imageName, v.path)
		return path.Join(blobsPathPrefix...), nil
	default:
		return "", fmt.Errorf("unknown path spec: %#v", v)
	}
}
