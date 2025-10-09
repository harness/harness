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
	"fmt"
	"strings"
)

var (
	// ErrNotFound is returned in case resources related to a githook call aren't found.
	ErrNotFound = fmt.Errorf("not found")
)

// Client is an abstraction of a githook client that can be used to trigger githook calls.
type Client interface {
	PreReceive(ctx context.Context, in PreReceiveInput) (Output, error)
	Update(ctx context.Context, in UpdateInput) (Output, error)
	PostReceive(ctx context.Context, in PostReceiveInput) (Output, error)
}

// ClientFactory is an abstraction of a factory that creates a new client based on the provided environment variables.
type ClientFactory interface {
	NewClient(envVars map[string]string) (Client, error)
}

// TODO: move to single representation once we have our custom Git CLI wrapper.
func EnvVarsToMap(in []string) (map[string]string, error) {
	out := map[string]string{}
	for _, entry := range in {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("unexpected entry in input: %q", entry)
		}

		key = strings.TrimSpace(key)
		out[key] = value
	}

	return out, nil
}
