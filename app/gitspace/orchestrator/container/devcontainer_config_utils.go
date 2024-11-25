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
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/runarg"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func ExtractRunArgs(
	ctx context.Context,
	spaceID int64,
	runArgProvider runarg.Provider,
	runArgsRaw []string,
) (map[types.RunArg]*types.RunArgValue, error) {
	supportedRunArgsMap, err := runArgProvider.ProvideSupportedRunArgs(ctx, spaceID)
	if err != nil {
		return nil, err
	}

	var runArgsMap = make(map[types.RunArg]*types.RunArgValue)
	primaryLoopCounter := 0
	for primaryLoopCounter < len(runArgsRaw) {
		currentArg := runArgsRaw[primaryLoopCounter]
		if currentArg == "" || !isArg(currentArg) {
			primaryLoopCounter++
			continue
		}

		argParts := strings.SplitN(currentArg, "=", 2)
		argKey := argParts[0]

		currentRunArgDefinition, isSupportedArg := supportedRunArgsMap[types.RunArg(argKey)]
		if !isSupportedArg {
			primaryLoopCounter++
			continue
		}
		updatedPrimaryLoopCounter, allowedValues, isAnyValueBlocked := getValues(runArgsRaw, argParts,
			primaryLoopCounter, currentRunArgDefinition)

		primaryLoopCounter = updatedPrimaryLoopCounter

		if isAnyValueBlocked && len(allowedValues) == 0 {
			continue
		}

		currentRunArgValue := types.RunArgValue{
			Name:   currentRunArgDefinition.Name,
			Values: allowedValues,
		}

		existingRunArgValue, isAlreadyPresent := runArgsMap[currentRunArgDefinition.Name]
		if isAlreadyPresent && currentRunArgDefinition.AllowMultipleOccurences {
			existingRunArgValue.Values = append(existingRunArgValue.Values, currentRunArgValue.Values...)
		} else {
			runArgsMap[currentRunArgDefinition.Name] = &currentRunArgValue
		}
	}

	return runArgsMap, nil
}

func getValues(
	runArgs []string,
	argParts []string,
	primaryLoopCounter int,
	currentRunArgDefinition types.RunArgDefinition,
) (int, []string, bool) {
	values := make([]string, 0)
	if len(argParts) > 1 {
		values = append(values, argParts[1])
		primaryLoopCounter++
	} else {
		var secondaryLoopCounter = primaryLoopCounter + 1
		for secondaryLoopCounter < len(runArgs) {
			currentValue := runArgs[secondaryLoopCounter]
			if isArg(currentValue) {
				break
			}
			values = append(values, currentValue)
			secondaryLoopCounter++
		}
		primaryLoopCounter = secondaryLoopCounter
	}
	allowedValues, isAnyValueBlocked := filterAllowedValues(values, currentRunArgDefinition)
	return primaryLoopCounter, allowedValues, isAnyValueBlocked
}

func filterAllowedValues(values []string, currentRunArgDefinition types.RunArgDefinition) ([]string, bool) {
	isAnyValueBlocked := false
	allowedValues := make([]string, 0)
	for _, v := range values {
		switch {
		case len(currentRunArgDefinition.AllowedValues) > 0:
			if _, ok := currentRunArgDefinition.AllowedValues[v]; ok {
				allowedValues = append(allowedValues, v)
			} else {
				isAnyValueBlocked = true
			}
		case len(currentRunArgDefinition.BlockedValues) > 0:
			if _, ok := currentRunArgDefinition.BlockedValues[v]; !ok {
				allowedValues = append(allowedValues, v)
			} else {
				isAnyValueBlocked = true
			}
		default:
			allowedValues = append(allowedValues, v)
		}
	}
	return allowedValues, isAnyValueBlocked
}

func isArg(str string) bool {
	return strings.HasPrefix(str, "--") || strings.HasPrefix(str, "-")
}

func ExtractEnv(devcontainerConfig types.DevcontainerConfig, runArgsMap map[types.RunArg]*types.RunArgValue) []string {
	var env []string
	for key, value := range devcontainerConfig.ContainerEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	envFromRunArgs := getEnv(runArgsMap)
	env = append(env, envFromRunArgs...)
	return env
}

func ExtractForwardPorts(devcontainerConfig types.DevcontainerConfig) []int {
	var ports []int
	for _, strPort := range devcontainerConfig.ForwardPorts {
		portAsInt, err := strPort.Int64() // Using Atoi to convert string to int
		if err != nil {
			log.Warn().Msgf("Error converting port string '%s' to int: %v", strPort, err)
			continue // Skip the invalid port
		}
		ports = append(ports, int(portAsInt))
	}
	return ports
}

func ExtractLifecycleCommands(actionType PostAction, devcontainerConfig types.DevcontainerConfig) []string {
	switch actionType {
	case PostCreateAction:
		return devcontainerConfig.PostCreateCommand.ToCommandArray()
	case PostStartAction:
		return devcontainerConfig.PostStartCommand.ToCommandArray()
	default:
		return []string{} // Return empty string if actionType is not recognized
	}
}
