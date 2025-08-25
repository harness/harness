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

package ide

import (
	"fmt"

	"github.com/harness/gitness/types/enum"
)

type Factory struct {
	ides map[enum.IDEType]IDE
}

func NewFactory(
	vscode *VSCode,
	vscodeWeb *VSCodeWeb,
	jetBrainsIDEsMap map[enum.IDEType]*JetBrainsIDE,
	cursor *Cursor,
	windsurf *Windsurf,
) Factory {
	ides := make(map[enum.IDEType]IDE)
	ides[enum.IDETypeVSCode] = vscode
	ides[enum.IDETypeVSCodeWeb] = vscodeWeb
	ides[enum.IDETypeCursor] = cursor
	ides[enum.IDETypeWindsurf] = windsurf
	ides[enum.IDETypeIntelliJ] = jetBrainsIDEsMap[enum.IDETypeIntelliJ]
	ides[enum.IDETypePyCharm] = jetBrainsIDEsMap[enum.IDETypePyCharm]
	ides[enum.IDETypeGoland] = jetBrainsIDEsMap[enum.IDETypeGoland]
	ides[enum.IDETypeWebStorm] = jetBrainsIDEsMap[enum.IDETypeWebStorm]
	ides[enum.IDETypeCLion] = jetBrainsIDEsMap[enum.IDETypeCLion]
	ides[enum.IDETypePHPStorm] = jetBrainsIDEsMap[enum.IDETypePHPStorm]
	ides[enum.IDETypeRubyMine] = jetBrainsIDEsMap[enum.IDETypeRubyMine]
	ides[enum.IDETypeRider] = jetBrainsIDEsMap[enum.IDETypeRider]
	return Factory{ides: ides}
}

func NewFactoryWithIDEs(ides map[enum.IDEType]IDE) Factory {
	return Factory{ides: ides}
}

func (f *Factory) GetIDE(ideType enum.IDEType) (IDE, error) {
	val, exist := f.ides[ideType]
	if !exist {
		return nil, fmt.Errorf("unsupported IDE type: %s", ideType)
	}

	return val, nil
}
