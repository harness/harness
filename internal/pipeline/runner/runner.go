// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package runner

import (
	"github.com/harness/gitness/internal/pipeline/manager"
	"github.com/harness/gitness/types"

	"github.com/drone-runners/drone-runner-docker/engine"
	"github.com/drone-runners/drone-runner-docker/engine/compiler"
	"github.com/drone-runners/drone-runner-docker/engine/linter"
	"github.com/drone-runners/drone-runner-docker/engine/resource"
	compiler2 "github.com/drone-runners/drone-runner-docker/engine2/compiler"
	engine2 "github.com/drone-runners/drone-runner-docker/engine2/engine"
	runtime2 "github.com/drone-runners/drone-runner-docker/engine2/runtime"
	"github.com/drone/drone-go/drone"
	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/environ/provider"
	"github.com/drone/runner-go/pipeline/reporter/history"
	"github.com/drone/runner-go/pipeline/reporter/remote"
	"github.com/drone/runner-go/pipeline/runtime"
	"github.com/drone/runner-go/pipeline/uploader"
	"github.com/drone/runner-go/registry"
	"github.com/drone/runner-go/secret"
)

func NewExecutionRunner(
	config *types.Config,
	client runnerclient.Client,
	m manager.ExecutionManager,
) (*runtime2.Runner, error) {
	// For linux, containers need to have extra hosts set in order to interact with
	// the gitness container.
	extraHosts := []string{"host.docker.internal:host-gateway"}
	compiler := &compiler.Compiler{
		Environ:    provider.Static(map[string]string{}),
		Registry:   registry.Static([]*drone.Registry{}),
		Secret:     secret.Encrypted(),
		ExtraHosts: extraHosts,
	}

	remote := remote.New(client)
	upload := uploader.New(client)
	tracer := history.New(remote)
	engine, err := engine.NewEnv(engine.Opts{})
	if err != nil {
		return nil, err
	}

	exec := runtime.NewExecer(tracer, remote, upload,
		engine, int64(config.CI.ParallelWorkers))

	legacyRunner := &runtime.Runner{
		Machine:  config.InstanceID,
		Client:   client,
		Reporter: tracer,
		Lookup:   resource.Lookup,
		Lint:     linter.New().Lint,
		Compiler: compiler,
		Exec:     exec.Exec,
	}

	engine2, err := engine2.NewEnv(engine2.Opts{})
	if err != nil {
		return nil, err
	}

	exec2 := runtime2.NewExecer(tracer, remote, upload, engine2, int64(config.CI.ParallelWorkers))

	compiler2 := &compiler2.CompilerImpl{
		Environ:    provider.Static(map[string]string{}),
		Registry:   registry.Static([]*drone.Registry{}),
		Secret:     secret.Encrypted(),
		ExtraHosts: extraHosts,
	}

	runner := &runtime2.Runner{
		Machine:      config.InstanceID,
		Client:       client,
		Reporter:     tracer,
		Compiler:     compiler2,
		Exec:         exec2.Exec,
		LegacyRunner: legacyRunner,
	}

	return runner, nil
}
