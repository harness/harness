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

package hook

import (
	"context"
)

// NoopClient directly returns success with the provided messages, without any other impact.
type NoopClient struct {
	messages []string
}

func NewNoopClient(messages []string) Client {
	return &NoopClient{messages: messages}
}

func (c *NoopClient) PreReceive(_ context.Context, _ PreReceiveInput) (Output, error) {
	return Output{Messages: c.messages}, nil
}

func (c *NoopClient) Update(_ context.Context, _ UpdateInput) (Output, error) {
	return Output{Messages: c.messages}, nil
}

func (c *NoopClient) PostReceive(_ context.Context, _ PostReceiveInput) (Output, error) {
	return Output{Messages: c.messages}, nil
}
