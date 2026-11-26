// Copyright 2026 Harness, Inc.
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

package runner

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/pipeline/resolver"
	"github.com/harness/gitness/types"

	"github.com/drone/drone-go/drone"
	"github.com/stretchr/testify/require"
)

func TestInfNewExecutionRunner(t *testing.T) {
	cfg := &types.Config{InstanceID: "inst-1"}
	cfg.CI.ParallelWorkers = 2
	cfg.CI.ContainerNetworks = []string{"net1"}
	cfg.Docker.Host = "tcp://127.0.0.1:1"

	r, err := NewExecutionRunner(cfg, nil, resolver.NewManager(cfg, nil, nil, nil, nil))
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Equal(t, "inst-1", r.Machine)
	require.NotNil(t, r.LegacyRunner)
	require.NotNil(t, r.Compiler)
}

func TestInfNewExecutionPoller(t *testing.T) {
	// a nil runner panics inside Run; the poller's dispatch must turn that into an error.
	p := NewExecutionPoller(nil, nil)
	require.NotNil(t, p)
	require.Equal(t, "pipeline", p.Filter.Kind)
	require.Equal(t, "docker", p.Filter.Type)

	err := p.Dispatch(context.Background(), &drone.Stage{ID: 1})
	require.ErrorContains(t, err, "panic received")
}
