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

package usage

import (
	"github.com/harness/gitness/types"

	"github.com/alecthomas/units"
)

type Config struct {
	ChunkSize  int64
	MaxWorkers int
}

func NewConfig(global *types.Config) Config {
	var err error
	var n units.Base2Bytes
	cfg := Config{
		MaxWorkers: global.UsageMetrics.MaxWorkers,
	}

	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = 50
	}

	chunkSize := global.UsageMetrics.ChunkSize
	if chunkSize == "" {
		chunkSize = "10MiB"
	}

	n, err = units.ParseBase2Bytes(chunkSize)
	if err != nil {
		panic(err)
	}
	cfg.ChunkSize = int64(n)

	return cfg
}
