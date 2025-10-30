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
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideVSCodeWebService,
	ProvideVSCodeService,
	ProvideCursorService,
	ProvideWindsurfService,
	ProvideJetBrainsIDEsService,
	ProvideIDEFactory,
)

func ProvideVSCodeWebService(config *VSCodeWebConfig) *VSCodeWeb {
	return NewVsCodeWebService(config, "http")
}

func ProvideVSCodeService(config *VSCodeConfig) *VSCode {
	return NewVsCodeService(config)
}

func ProvideCursorService(config *CursorConfig) *Cursor {
	return NewCursorService(config)
}

func ProvideWindsurfService(config *WindsurfConfig) *Windsurf {
	return NewWindsurfService(config)
}

func ProvideJetBrainsIDEsService(config *JetBrainsIDEConfig) map[enum.IDEType]*JetBrainsIDE {
	return map[enum.IDEType]*JetBrainsIDE{
		enum.IDETypeIntelliJ: NewJetBrainsIDEService(config, enum.IDETypeIntelliJ),
		enum.IDETypePyCharm:  NewJetBrainsIDEService(config, enum.IDETypePyCharm),
		enum.IDETypeGoland:   NewJetBrainsIDEService(config, enum.IDETypeGoland),
		enum.IDETypeWebStorm: NewJetBrainsIDEService(config, enum.IDETypeWebStorm),
		enum.IDETypeCLion:    NewJetBrainsIDEService(config, enum.IDETypeCLion),
		enum.IDETypePHPStorm: NewJetBrainsIDEService(config, enum.IDETypePHPStorm),
		enum.IDETypeRubyMine: NewJetBrainsIDEService(config, enum.IDETypeRubyMine),
		enum.IDETypeRider:    NewJetBrainsIDEService(config, enum.IDETypeRider),
	}
}

func ProvideIDEFactory(
	vscode *VSCode,
	vscodeWeb *VSCodeWeb,
	jetBrainsIDEsMap map[enum.IDEType]*JetBrainsIDE,
	cursor *Cursor,
	windsurf *Windsurf,
) Factory {
	return NewFactory(vscode, vscodeWeb, jetBrainsIDEsMap, cursor, windsurf)
}
