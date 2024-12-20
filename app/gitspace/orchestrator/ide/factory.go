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

func NewFactory(vscode *VSCode, vscodeWeb *VSCodeWeb) Factory {
	ides := make(map[enum.IDEType]IDE)
	ides[enum.IDETypeVSCode] = vscode
	ides[enum.IDETypeVSCodeWeb] = vscodeWeb
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
