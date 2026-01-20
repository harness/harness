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
	"time"

	"github.com/harness/gitness/types"
)

// MinFlushInterval defines the minimum allowed flush interval.
// This can be overridden in tests for faster execution.
var MinFlushInterval = time.Minute

type Config struct {
	FlushInterval time.Duration
}

func (c *Config) Sanitize() {
	if c.FlushInterval < MinFlushInterval {
		c.FlushInterval = MinFlushInterval
	}
}

func NewConfig(global *types.Config) Config {
	cfg := Config{
		FlushInterval: global.UsageMetrics.FlushInterval,
	}

	cfg.Sanitize()

	return cfg
}
