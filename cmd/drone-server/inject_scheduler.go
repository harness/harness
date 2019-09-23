// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/drone/drone/cmd/drone-server/config"
	"github.com/drone/drone/core"
	"github.com/drone/drone/scheduler/kube"
	"github.com/drone/drone/scheduler/nomad"
	"github.com/drone/drone/scheduler/queue"

	"github.com/google/wire"
	"github.com/sirupsen/logrus"
)

// wire set for loading the scheduler.
var schedulerSet = wire.NewSet(
	provideScheduler,
)

// provideScheduler is a Wire provider function that returns a
// scheduler based on the environment configuration.
func provideScheduler(store core.StageStore, config config.Config) core.Scheduler {
	switch {
	case config.Kube.Enabled:
		return provideKubernetesScheduler(config)
	case config.Nomad.Enabled:
		return provideNomadScheduler(config)
	default:
		return provideQueueScheduler(store, config)
	}
}

// provideKubernetesScheduler is a Wire provider function that
// returns a nomad kubernetes from the environment configuration.
func provideKubernetesScheduler(config config.Config) core.Scheduler {
	logrus.Info("main: kubernetes scheduler enabled")
	sched, err := kube.FromConfig(kube.Config{
		Namespace:       config.Kube.Namespace,
		ServiceAccount:  config.Kube.ServiceAccountName,
		ConfigURL:       config.Kube.URL,
		ConfigPath:      config.Kube.Path,
		TTL:             config.Kube.TTL,
		Image:           config.Kube.Image,
		ImagePullPolicy: config.Kube.PullPolicy,
		ImagePrivileged: config.Runner.Privileged,
		// LimitMemory:      config.Nomad.Memory,
		// LimitCompute:     config.Nomad.CPU,
		// RequestMemory:    config.Nomad.Memory,
		// RequestCompute:   config.Nomad.CPU,
		CallbackHost:     config.RPC.Host,
		CallbackProto:    config.RPC.Proto,
		CallbackSecret:   config.RPC.Secret,
		SecretToken:      config.Secrets.Password,
		SecretEndpoint:   config.Secrets.Endpoint,
		SecretInsecure:   config.Secrets.SkipVerify,
		RegistryToken:    config.Registries.Password,
		RegistryEndpoint: config.Registries.Endpoint,
		RegistryInsecure: config.Registries.SkipVerify,
		LogDebug:         config.Logging.Debug,
		LogTrace:         config.Logging.Trace,
		LogPretty:        config.Logging.Pretty,
		LogText:          config.Logging.Text,
	})
	if err != nil {
		logrus.WithError(err).
			Fatalln("main: cannot create kubernetes client")
	}
	return sched
}

// provideNomadScheduler is a Wire provider function that returns
// a nomad scheduler from the environment configuration.
func provideNomadScheduler(config config.Config) core.Scheduler {
	logrus.Info("main: nomad scheduler enabled")
	sched, err := nomad.FromConfig(nomad.Config{
		Datacenter:      config.Nomad.Datacenters,
		Namespace:       config.Nomad.Namespace,
		Region:          config.Nomad.Region,
		DockerImage:     config.Nomad.Image,
		DockerImagePull: config.Nomad.ImagePull,
		DockerImagePriv: config.Runner.Privileged,
		DockerHost:      "",
		DockerHostWin:   "",
		// LimitMemory:      config.Nomad.Memory,
		// LimitCompute:     config.Nomad.CPU,
		RequestMemory:    config.Nomad.Memory,
		RequestCompute:   config.Nomad.CPU,
		CallbackHost:     config.RPC.Host,
		CallbackProto:    config.RPC.Proto,
		CallbackSecret:   config.RPC.Secret,
		SecretToken:      config.Secrets.Password,
		SecretEndpoint:   config.Secrets.Endpoint,
		SecretInsecure:   config.Secrets.SkipVerify,
		RegistryToken:    config.Registries.Password,
		RegistryEndpoint: config.Registries.Endpoint,
		RegistryInsecure: config.Registries.SkipVerify,
		LogDebug:         config.Logging.Debug,
		LogTrace:         config.Logging.Trace,
		LogPretty:        config.Logging.Pretty,
		LogText:          config.Logging.Text,
	})
	if err != nil {
		logrus.WithError(err).
			Fatalln("main: cannot create nomad client")
	}
	return sched
}

// provideQueueScheduler is a Wire provider function that
// returns an in-memory scheduler for use by the built-in
// docker runner, and by remote agents.
func provideQueueScheduler(store core.StageStore, config config.Config) core.Scheduler {
	logrus.Info("main: internal scheduler enabled")
	return queue.New(store)
}
