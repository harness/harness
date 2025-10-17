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

package types

import (
	"github.com/rs/zerolog"
)

// NewZerologAdapter creates a new adapter from a zerolog.Logger.
func NewZerologAdapter(logger *zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger: logger}
}

// Implement the Logger interface for ZerologAdapter.
func (z *ZerologAdapter) Info(msg string) {
	z.logger.Info().Msg("INFO: " + msg)
}

func (z *ZerologAdapter) Debug(msg string) {
	z.logger.Debug().Msg("DEBUG: " + msg)
}

func (z *ZerologAdapter) Warn(msg string) {
	z.logger.Warn().Msg("WARN: " + msg)
}

func (z *ZerologAdapter) Error(msg string, err error) {
	z.logger.Err(err).Msg("ERROR: " + msg)
}
