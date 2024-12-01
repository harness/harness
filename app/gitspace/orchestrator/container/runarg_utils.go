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

package container

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/types"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-units"
	"github.com/rs/zerolog/log"
)

func getHostResources(runArgsMap map[types.RunArg]*types.RunArgValue) (container.Resources, error) { // nolint: gocognit
	var resources = container.Resources{}
	blkioWeight, err := getArgValueUint16(runArgsMap, types.RunArgBlkioWeight)
	if err != nil {
		return resources, err
	}
	resources.BlkioWeight = blkioWeight

	cpuShares, err := getArgValueInt64(runArgsMap, types.RunArgCPUShares)
	if err != nil {
		return resources, err
	}
	resources.CPUShares = cpuShares

	memory, err := getArgValueMemoryBytes(runArgsMap, types.RunArgMemory)
	if err != nil {
		return resources, err
	}
	resources.Memory = memory

	cpus, err := getCPUs(runArgsMap, types.RunArgCpus)
	if err != nil {
		return resources, err
	}
	resources.NanoCPUs = cpus

	resources.CgroupParent = getArgValueString(runArgsMap, types.RunArgCgroupParent)

	cpuPeriod, err := getArgValueInt64(runArgsMap, types.RunArgCPUPeriod)
	if err != nil {
		return resources, err
	}
	resources.CPUPeriod = cpuPeriod

	cpuQuota, err := getArgValueInt64(runArgsMap, types.RunArgCPUQuota)
	if err != nil {
		return resources, err
	}
	resources.CPUQuota = cpuQuota

	cpuRTPeriod, err := getArgValueInt64(runArgsMap, types.RunArgCPURtPeriod)
	if err != nil {
		return resources, err
	}
	resources.CPURealtimePeriod = cpuRTPeriod

	cpuRTRuntime, err := getArgValueInt64(runArgsMap, types.RunArgCPURtRuntime)
	if err != nil {
		return resources, err
	}
	resources.CPURealtimeRuntime = cpuRTRuntime

	resources.CpusetCpus = getArgValueString(runArgsMap, types.RunArgCpusetCpus)

	resources.CpusetMems = getArgValueString(runArgsMap, types.RunArgCpusetMems)

	cpuCount, err := getArgValueInt64(runArgsMap, types.RunArgCPUCount)
	if err != nil {
		return resources, err
	}
	resources.CPUCount = cpuCount

	cpuPercent, err := getArgValueInt64(runArgsMap, types.RunArgCPUPercent)
	if err != nil {
		return resources, err
	}
	resources.CPUPercent = cpuPercent

	kernelMemory, err := getArgValueMemoryBytes(runArgsMap, types.RunArgKernelMemory)
	if err != nil {
		return resources, err
	}
	resources.KernelMemory = kernelMemory

	memoryReservation, err := getArgValueMemoryBytes(runArgsMap, types.RunArgMemoryReservation)
	if err != nil {
		return resources, err
	}
	resources.MemoryReservation = memoryReservation

	memorySwappiness, err := getArgValueInt64Ptr(runArgsMap, types.RunArgMemorySwappiness)
	if err != nil {
		return resources, err
	}
	resources.MemorySwappiness = memorySwappiness

	memorySwap, err := getArgValueMemorySwapBytes(runArgsMap, types.RunArgMemorySwap)
	if err != nil {
		return resources, err
	}
	resources.MemorySwap = memorySwap

	resources.OomKillDisable = getArgValueBoolPtr(runArgsMap, types.RunArgOomKillDisable)

	pidsLimit, err := getArgValueInt64Ptr(runArgsMap, types.RunArgPidsLimit)
	if err != nil {
		return resources, err
	}
	resources.PidsLimit = pidsLimit

	ioMaxiops, err := getArgValueUint64(runArgsMap, types.RunArgIoMaxiops)
	if err != nil {
		return resources, err
	}
	resources.IOMaximumIOps = ioMaxiops

	ioMaxbandwidth, err := getArgValueMemoryBytes(runArgsMap, types.RunArgIoMaxbandwidth)
	if err != nil {
		return resources, err
	}
	resources.IOMaximumBandwidth = uint64(ioMaxbandwidth)

	if arg, ok := runArgsMap[types.RunArgUlimit]; ok {
		ulimits := []*container.Ulimit{}
		for _, v := range arg.Values {
			ulimit, err := units.ParseUlimit(v)
			if err != nil {
				return resources, err
			}
			ulimits = append(ulimits, ulimit)
		}
		resources.Ulimits = ulimits
	}
	return resources, nil
}

func getNetworkMode(runArgsMap map[types.RunArg]*types.RunArgValue) container.NetworkMode {
	return container.NetworkMode(getArgValueString(runArgsMap, types.RunArgNetwork))
}

func getCapDrop(runArgsMap map[types.RunArg]*types.RunArgValue) strslice.StrSlice {
	return getArgValueStringSlice(runArgsMap, types.RunArgCapDrop)
}

func getSHMSize(runArgsMap map[types.RunArg]*types.RunArgValue) (int64, error) {
	return getArgValueMemoryBytes(runArgsMap, types.RunArgShmSize)
}

func getArgValueMemorySwapBytes(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (int64, error) {
	value := getArgValueString(runArgsMap, argName)
	if value == "" {
		return 0, nil
	}
	if value == "-1" {
		return -1, nil
	}
	memorySwapBytes, err := units.RAMInBytes(value)
	if err != nil {
		return 0, err
	}
	return memorySwapBytes, nil
}

func getSysctls(runArgsMap map[types.RunArg]*types.RunArgValue) map[string]string {
	values := getArgValueStringSlice(runArgsMap, types.RunArgSysctl)
	var opt = map[string]string{}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			parts = append(parts, "")
		}
		opt[parts[0]] = parts[1]
	}
	return opt
}

func getDNS(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgDNS)
}

func getDNSOptions(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgDNSOption)
}

func getDNSSearch(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgDNSSearch)
}

func getCgroupNSMode(runArgsMap map[types.RunArg]*types.RunArgValue) container.CgroupnsMode {
	value := getArgValueString(runArgsMap, types.RunArgCgroupns)
	return container.CgroupnsMode(value)
}

func getIPCMode(runArgsMap map[types.RunArg]*types.RunArgValue) container.IpcMode {
	return container.IpcMode(getArgValueString(runArgsMap, types.RunArgIpc))
}

func getIsolation(runArgsMap map[types.RunArg]*types.RunArgValue) container.Isolation {
	return container.Isolation(getArgValueString(runArgsMap, types.RunArgIsolation))
}

func getRuntime(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	return getArgValueString(runArgsMap, types.RunArgRuntime)
}

func getPlatform(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	return getArgValueString(runArgsMap, types.RunArgPlatform)
}

func getPIDMode(runArgsMap map[types.RunArg]*types.RunArgValue) container.PidMode {
	return container.PidMode(getArgValueString(runArgsMap, types.RunArgPid))
}

func getSecurityOpt(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgSecurityOpt)
}

func getStorageOpt(runArgsMap map[types.RunArg]*types.RunArgValue) map[string]string {
	values := getArgValueStringSlice(runArgsMap, types.RunArgStorageOpt)
	var opt = map[string]string{}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			parts = append(parts, "")
		}
		opt[parts[0]] = parts[1]
	}
	return opt
}

func getAutoRemove(runArgsMap map[types.RunArg]*types.RunArgValue) bool {
	return getArgValueBool(runArgsMap, types.RunArgRm)
}

func getInit(runArgsMap map[types.RunArg]*types.RunArgValue) *bool {
	return getArgValueBoolPtr(runArgsMap, types.RunArgInit)
}

func getEnv(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgEnv)
}

func getLinks(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgLink)
}

func getOomScoreAdj(runArgsMap map[types.RunArg]*types.RunArgValue) (int, error) {
	return getArgValueInt(runArgsMap, types.RunArgOomScoreAdj)
}

func getRestartPolicy(runArgsMap map[types.RunArg]*types.RunArgValue) (container.RestartPolicy, error) {
	value := getArgValueString(runArgsMap, types.RunArgRestart)
	if len(value) == 0 {
		return container.RestartPolicy{}, nil
	}
	parts := strings.SplitN(value, ":", 2)
	maxCount := 0
	if container.RestartPolicyMode(parts[0]) == container.RestartPolicyOnFailure && len(parts) == 2 {
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			return container.RestartPolicy{}, err
		}
		maxCount = count
	}
	return container.RestartPolicy{
		Name:              container.RestartPolicyMode(parts[0]),
		MaximumRetryCount: maxCount,
	}, nil
}

func getExtraHosts(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgAddHost)
}

func getHostname(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	return getArgValueString(runArgsMap, types.RunArgHostname)
}

func getDomainname(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	return getArgValueString(runArgsMap, types.RunArgDomainname)
}

func getMACAddress(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	return getArgValueString(runArgsMap, types.RunArgMacAddress)
}

func getStopSignal(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	return getArgValueString(runArgsMap, types.RunArgStopSignal)
}

func getStopTimeout(runArgsMap map[types.RunArg]*types.RunArgValue) (*int, error) {
	return getArgValueIntPtr(runArgsMap, types.RunArgStopTimeout)
}

func getImagePullPolicy(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	policy := getArgValueString(runArgsMap, types.RunArgPull)
	if policy == "" {
		policy = "missing"
	}
	return policy
}

func getUser(runArgsMap map[types.RunArg]*types.RunArgValue) string {
	return getArgValueString(runArgsMap, types.RunArgUser)
}

func getEntrypoint(runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	return getArgValueStringSlice(runArgsMap, types.RunArgEntrypoint)
}

func getHealthCheckConfig(runArgsMap map[types.RunArg]*types.RunArgValue) (*container.HealthConfig, error) {
	var healthConfig = &container.HealthConfig{}

	retries, err := getArgValueInt(runArgsMap, types.RunArgHealthRetries)
	if err != nil {
		return healthConfig, err
	}
	healthConfig.Retries = retries

	interval, err := getArgValueDuration(runArgsMap, types.RunArgHealthInterval)
	if err != nil {
		return healthConfig, err
	}
	healthConfig.Interval = interval

	timeout, err := getArgValueDuration(runArgsMap, types.RunArgHealthTimeout)
	if err != nil {
		return healthConfig, err
	}
	healthConfig.Timeout = timeout

	startPeriod, err := getArgValueDuration(runArgsMap, types.RunArgHealthStartPeriod)
	if err != nil {
		return healthConfig, err
	}
	healthConfig.StartPeriod = startPeriod

	startInterval, err := getArgValueDuration(runArgsMap, types.RunArgHealthStartInterval)
	if err != nil {
		return healthConfig, err
	}
	healthConfig.StartInterval = startInterval

	if _, ok := runArgsMap[types.RunArgNoHealthcheck]; ok {
		healthConfig.Test = []string{"NONE"}
	} else if arg, healthCmdOK := runArgsMap[types.RunArgHealthCmd]; healthCmdOK {
		healthConfig.Test = arg.Values
	}

	return healthConfig, nil
}

func getLabels(runArgsMap map[types.RunArg]*types.RunArgValue) map[string]string {
	labelsMap := make(map[string]string)
	arg, ok := runArgsMap[types.RunArgLabel]
	if ok {
		labels := arg.Values
		for _, v := range labels {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) < 2 {
				labelsMap[parts[0]] = ""
			} else {
				labelsMap[parts[0]] = parts[1]
			}
		}
	}
	return labelsMap
}

func getAnnotations(runArgsMap map[types.RunArg]*types.RunArgValue) map[string]string {
	arg, ok := runArgsMap[types.RunArgAnnotation]
	annotationsMap := make(map[string]string)
	if ok {
		annotations := arg.Values
		for _, v := range annotations {
			annotationParts := strings.SplitN(v, "=", 2)
			if len(annotationParts) != 2 {
				log.Warn().Msgf("invalid annotation: %s", v)
			} else {
				annotationsMap[annotationParts[0]] = annotationParts[1]
			}
		}
	}
	return annotationsMap
}

func getArgValueMemoryBytes(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (int64, error) {
	value := getArgValueString(runArgsMap, argName)
	if value == "" {
		return 0, nil
	}
	memoryBytes, err := units.RAMInBytes(value)
	if err != nil {
		return 0, err
	}
	return memoryBytes, nil
}

func getArgValueString(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) string {
	if arg, ok := runArgsMap[argName]; ok {
		return arg.Values[0]
	}
	return ""
}

func getArgValueStringSlice(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) []string {
	if arg, ok := runArgsMap[argName]; ok {
		return arg.Values
	}
	return []string{}
}

func getArgValueInt(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (int, error) {
	value, err := getArgValueIntPtr(runArgsMap, argName)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return 0, nil
	}
	return *value, nil
}

func getArgValueIntPtr(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (*int, error) {
	if arg, ok := runArgsMap[argName]; ok {
		value, err := strconv.Atoi(arg.Values[0])
		if err != nil {
			return nil, err
		}
		return &value, nil
	}
	return nil, nil // nolint: nilnil
}

func getArgValueUint16(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (uint16, error) {
	if arg, ok := runArgsMap[argName]; ok {
		value, err := strconv.ParseUint(arg.Values[0], 10, 16)
		if err != nil {
			return 0, err
		}
		return uint16(value), nil
	}
	return 0, nil
}

func getArgValueUint64(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (uint64, error) {
	if arg, ok := runArgsMap[argName]; ok {
		value, err := strconv.ParseUint(arg.Values[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return value, nil
	}
	return 0, nil
}

func getCPUs(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (int64, error) {
	if arg, ok := runArgsMap[argName]; ok {
		value := arg.Values[0]
		cpu, ok1 := new(big.Rat).SetString(value) // nolint: gosec
		if !ok1 {
			return 0, fmt.Errorf("failed to parse %v as a rational number", value)
		}
		nano := cpu.Mul(cpu, big.NewRat(1e9, 1))
		if !nano.IsInt() {
			return 0, fmt.Errorf("value is too precise")
		}
		return nano.Num().Int64(), nil
	}
	return 0, nil
}

func getArgValueInt64(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (int64, error) {
	value, err := getArgValueInt64Ptr(runArgsMap, argName)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return 0, nil
	}
	return *value, nil
}

func getArgValueInt64Ptr(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (*int64, error) {
	if arg, ok := runArgsMap[argName]; ok {
		value, err := strconv.ParseInt(arg.Values[0], 10, 64)
		if err != nil {
			return nil, err
		}
		return &value, nil
	}
	return nil, nil // nolint: nilnil
}

func getArgValueDuration(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) (time.Duration, error) {
	defaultDur := time.Second * 0
	if arg, ok := runArgsMap[argName]; ok {
		dur, err := time.ParseDuration(arg.Values[0])
		if err != nil {
			return defaultDur, err
		}
		return dur, nil
	}
	return defaultDur, nil
}

func getArgValueBool(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) bool {
	value := getArgValueBoolPtr(runArgsMap, argName)
	if value == nil {
		return false
	}
	return *value
}
func getArgValueBoolPtr(runArgsMap map[types.RunArg]*types.RunArgValue, argName types.RunArg) *bool {
	_, ok := runArgsMap[argName]
	if ok {
		return &ok
	}
	return nil
}
