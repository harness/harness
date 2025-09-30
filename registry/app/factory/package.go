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

package factory

import (
	"github.com/harness/gitness/registry/app/api/interfaces"
)

type PackageFactory interface {
	Register(helper interfaces.PackageHelper)
	Get(packageType string) interfaces.PackageHelper
	GetAllPackageTypes() []string
	IsValidPackageType(packageType string) bool
}

type packageFactory struct {
	factory map[string]interfaces.PackageHelper
}

func NewPackageFactory() PackageFactory {
	var factory = make(map[string]interfaces.PackageHelper)
	return &packageFactory{
		factory: factory,
	}
}

func (f *packageFactory) Register(helper interfaces.PackageHelper) {
	packageType := helper.GetPackageType()
	if _, ok := f.factory[packageType]; !ok {
		f.factory[packageType] = helper
	}
}

func (f *packageFactory) Get(packageType string) interfaces.PackageHelper {
	if _, ok := f.factory[packageType]; !ok {
		return nil
	}
	return f.factory[packageType]
}

func (f *packageFactory) GetAllPackageTypes() []string {
	var packageTypes []string
	for packageType := range f.factory {
		packageTypes = append(packageTypes, packageType)
	}
	return packageTypes
}

func (f *packageFactory) IsValidPackageType(packageType string) bool {
	if packageType == "" {
		return true
	}
	_, ok := f.factory[packageType]
	return ok
}
