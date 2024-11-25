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

package runarg

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"

	"gopkg.in/yaml.v3"

	_ "embed"
)

var _ Provider = (*StaticProvider)(nil)

//go:embed runArgs.yaml
var supportedRunArgsRaw []byte

type StaticProvider struct {
	supportedRunArgsMap map[types.RunArg]types.RunArgDefinition
}

func NewStaticProvider() (Provider, error) {
	allRunArgs := make([]types.RunArgDefinition, 0)
	err := yaml.Unmarshal(supportedRunArgsRaw, &allRunArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal runArgs.yaml: %w", err)
	}
	argsMap := make(map[types.RunArg]types.RunArgDefinition)
	for _, arg := range allRunArgs {
		if arg.Supported {
			argsMap[arg.Name] = arg
			if arg.ShortHand != "" {
				argsMap[arg.ShortHand] = arg
			}
		}
	}
	return &StaticProvider{supportedRunArgsMap: argsMap}, nil
}

// ProvideSupportedRunArgs provides a static map of supported run args.
func (s *StaticProvider) ProvideSupportedRunArgs(
	_ context.Context,
	_ int64,
) (map[types.RunArg]types.RunArgDefinition, error) {
	return s.supportedRunArgsMap, nil
}
