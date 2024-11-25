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
	"strings"
)

type RunArg string

const (
	RunArgAddHost             = RunArg("--add-host")
	RunArgAnnotation          = RunArg("--annotation")
	RunArgBlkioWeight         = RunArg("--blkio-weight")
	RunArgCapDrop             = RunArg("--cap-drop")
	RunArgCgroupParent        = RunArg("--cgroup-parent")
	RunArgCgroupns            = RunArg("--cgroupns")
	RunArgCPUCount            = RunArg("--cpu-count")
	RunArgCPUPercent          = RunArg("--cpu-percent")
	RunArgCPUPeriod           = RunArg("--cpu-period")
	RunArgCPUQuota            = RunArg("--cpu-quota")
	RunArgCPURtPeriod         = RunArg("--cpu-rt-period")
	RunArgCPURtRuntime        = RunArg("--cpu-rt-runtime")
	RunArgCPUShares           = RunArg("--cpu-shares")
	RunArgCpus                = RunArg("--cpus")
	RunArgCpusetCpus          = RunArg("--cpuset-cpus")
	RunArgCpusetMems          = RunArg("--cpuset-mems")
	RunArgDNS                 = RunArg("--dns")
	RunArgDNSOption           = RunArg("--dns-option")
	RunArgDNSSearch           = RunArg("--dns-search")
	RunArgDomainname          = RunArg("--domainname")
	RunArgEntrypoint          = RunArg("--entrypoint")
	RunArgEnv                 = RunArg("--env")
	RunArgHealthCmd           = RunArg("--health-cmd")
	RunArgHealthInterval      = RunArg("--health-interval")
	RunArgHealthRetries       = RunArg("--health-retries")
	RunArgHealthStartInterval = RunArg("--health-start-interval")
	RunArgHealthStartPeriod   = RunArg("--health-start-period")
	RunArgHealthTimeout       = RunArg("--health-timeout")
	RunArgHostname            = RunArg("--hostname")
	RunArgInit                = RunArg("--init")
	RunArgIoMaxbandwidth      = RunArg("--io-maxbandwidth")
	RunArgIoMaxiops           = RunArg("--io-maxiops")
	RunArgIpc                 = RunArg("--ipc")
	RunArgIsolation           = RunArg("--isolation")
	RunArgKernelMemory        = RunArg("--kernel-memory")
	RunArgLabel               = RunArg("--label")
	RunArgLink                = RunArg("--link")
	RunArgMacAddress          = RunArg("--mac-address")
	RunArgMemory              = RunArg("--memory")
	RunArgMemoryReservation   = RunArg("--memory-reservation")
	RunArgMemorySwap          = RunArg("--memory-swap")
	RunArgMemorySwappiness    = RunArg("--memory-swappiness")
	RunArgNetwork             = RunArg("--network")
	RunArgNoHealthcheck       = RunArg("--no-healthcheck")
	RunArgOomKillDisable      = RunArg("--oom-kill-disable")
	RunArgOomScoreAdj         = RunArg("--oom-score-adj")
	RunArgPid                 = RunArg("--pid")
	RunArgPidsLimit           = RunArg("--pids-limit")
	RunArgPlatform            = RunArg("--platform")
	RunArgPull                = RunArg("--pull")
	RunArgRestart             = RunArg("--restart")
	RunArgRm                  = RunArg("--rm")
	RunArgRuntime             = RunArg("--runtime")
	RunArgSecurityOpt         = RunArg("--security-opt")
	RunArgShmSize             = RunArg("--shm-size")
	RunArgStopSignal          = RunArg("--stop-signal")
	RunArgStopTimeout         = RunArg("--stop-timeout")
	RunArgStorageOpt          = RunArg("--storage-opt")
	RunArgSysctl              = RunArg("--sysctl")
	RunArgUlimit              = RunArg("--ulimit")
	RunArgUser                = RunArg("--user")
)

type RunArgDefinition struct {
	Name                    RunArg          `yaml:"name"`
	ShortHand               RunArg          `yaml:"short_hand"`
	Supported               bool            `yaml:"supported"`
	BlockedValues           map[string]bool `yaml:"blocked_values"`
	AllowedValues           map[string]bool `yaml:"allowed_values"`
	AllowMultipleOccurences bool            `yaml:"allow_multiple_occurrences"`
}

type RunArgValue struct {
	Name   RunArg
	Values []string
}

func (c *RunArgValue) String() string {
	return strings.Join(c.Values, ", ")
}
