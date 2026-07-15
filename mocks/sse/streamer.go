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

package sse

import (
	"context"

	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
)

type Streamer struct{ mock.Mock }

var _ sse.Streamer = (*Streamer)(nil)

func (m *Streamer) Publish(_ context.Context, spaceID int64, eventType enum.SSEType, data any) {
	m.Called(spaceID, eventType, data)
}

func (m *Streamer) Stream(
	_ context.Context,
	spaceID int64,
) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	args := m.Called(spaceID)
	ch, _ := args.Get(0).(<-chan *sse.Event)
	errCh, _ := args.Get(1).(<-chan error)
	fn, _ := args.Get(2).(func(context.Context) error)
	return ch, errCh, fn
}
