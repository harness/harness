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

package enum

import (
	"encoding/json"
	"fmt"
)

type IDEType string

func (i *IDEType) Enum() []interface{} {
	if i == nil {
		return nil
	}
	return toInterfaceSlice(ideTypes)
}

func (i *IDEType) String() string {
	if i == nil {
		return ""
	}
	return string(*i)
}

var ideTypes = []IDEType{IDETypeVSCode, IDETypeVSCodeWeb, IDETypeIntelliJ, IDETypePyCharm, IDETypeGoland,
	IDETypeWebStorm, IDETypeCLion, IDETypePHPStorm, IDETypeRubyMine, IDETypeRider}

var jetBrainsIDESet = map[IDEType]struct{}{
	IDETypeIntelliJ: {},
	IDETypePyCharm:  {},
	IDETypeGoland:   {},
	IDETypeWebStorm: {},
	IDETypeCLion:    {},
	IDETypePHPStorm: {},
	IDETypeRubyMine: {},
	IDETypeRider:    {},
}

const (
	IDETypeVSCode    IDEType = "vs_code"
	IDETypeVSCodeWeb IDEType = "vs_code_web"
	// all jetbrains IDEs.
	IDETypeIntelliJ IDEType = "intellij"
	IDETypePyCharm  IDEType = "pycharm"
	IDETypeGoland   IDEType = "goland"
	IDETypeWebStorm IDEType = "webstorm"
	IDETypeCLion    IDEType = "clion"
	IDETypePHPStorm IDEType = "phpstorm"
	IDETypeRubyMine IDEType = "rubymine"
	IDETypeRider    IDEType = "rider"
)

func IsJetBrainsIDE(t IDEType) bool {
	_, exist := jetBrainsIDESet[t]
	return exist
}

func (t *IDEType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	for _, v := range ideTypes {
		if IDEType(s) == v {
			*t = v
			return nil
		}
	}
	return fmt.Errorf("invalid IDEType: %s", s)
}
