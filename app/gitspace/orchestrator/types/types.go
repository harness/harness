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
	"context"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"

	"github.com/rs/zerolog"
)

type GitspaceLogger interface {
	Info(msg string)
	Debug(msg string)
	Error(msg string, err error)
}

// Step represents a single setup action.
type Step struct {
	Name          string
	Execute       func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger GitspaceLogger) error
	StopOnFailure bool // Flag to control whether execution should stop on failure
}

type ZerologAdapter struct {
	logger *zerolog.Logger
}
