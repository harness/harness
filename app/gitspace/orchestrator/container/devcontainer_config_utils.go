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
	"regexp"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/orchestrator/runarg"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

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
	argLoopCounter := 0
	for argLoopCounter < len(runArgsRaw) {
		currentArg := runArgsRaw[argLoopCounter]
		if currentArg == "" || !isArg(currentArg) {
			argLoopCounter++
			continue
		}

		argParts := strings.SplitN(currentArg, "=", 2)
		argKey := argParts[0]

		currentRunArgDefinition, isSupportedArg := supportedRunArgsMap[types.RunArg(argKey)]
		if !isSupportedArg {
			argLoopCounter++
			continue
		}
		updatedArgLoopCounter, allowedValues, isAnyValueBlocked := getValues(runArgsRaw, argParts, argLoopCounter,
			currentRunArgDefinition)

		argLoopCounter = updatedArgLoopCounter

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
	argLoopCounter int,
	currentRunArgDefinition types.RunArgDefinition,
) (int, []string, bool) {
	values := make([]string, 0)
	if len(argParts) > 1 {
		values = append(values, strings.TrimSpace(argParts[1]))
		argLoopCounter++
	} else {
		var valueLoopCounter = argLoopCounter + 1
		for valueLoopCounter < len(runArgs) {
			currentValue := runArgs[valueLoopCounter]
			if isArg(currentValue) {
				break
			}
			values = append(values, strings.TrimSpace(currentValue))
			valueLoopCounter++
		}
		argLoopCounter = valueLoopCounter
	}
	allowedValues, isAnyValueBlocked := filterAllowedValues(values, currentRunArgDefinition)
	return argLoopCounter, allowedValues, isAnyValueBlocked
}

func filterAllowedValues(
	values []string,
	currentRunArgDefinition types.RunArgDefinition,
) ([]string, bool) {
	isAnyValueBlocked := false
	allowedValues := make([]string, 0)
	for _, v := range values {
		switch {
		case len(currentRunArgDefinition.AllowedValues) > 0:
			for allowedValue := range currentRunArgDefinition.AllowedValues {
				matches, err := regexp.MatchString(allowedValue, v)
				if err != nil {
					log.Warn().Err(err).Msgf("error checking allowed values for RunArg %s value %s",
						currentRunArgDefinition.Name, v)
					continue
				}
				if matches {
					allowedValues = append(allowedValues, v)
				} else {
					log.Warn().Msgf("Value %s for runArg %s not allowed", v, currentRunArgDefinition.Name)
					isAnyValueBlocked = true
				}
			}
		case len(currentRunArgDefinition.BlockedValues) > 0:
			var isValueBlocked = false
			for blockedValue := range currentRunArgDefinition.BlockedValues {
				matches, err := regexp.MatchString(blockedValue, v)
				if err != nil {
					log.Warn().Err(err).Msgf("error checking blocked values for RunArg %s value %s",
						currentRunArgDefinition.Name, v)
					continue
				}
				if matches {
					log.Warn().Msgf("Value %s for runArg %s not allowed", v, currentRunArgDefinition.Name)
					isValueBlocked = true
					isAnyValueBlocked = true
				}
			}
			if !isValueBlocked {
				allowedValues = append(allowedValues, v)
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

func ExtractIDECustomizations(
	ideService ide.IDE,
	devcontainerConfig types.DevcontainerConfig,
) map[string]interface{} {
	var args = make(map[string]interface{})
	if ideService.Type() == enum.IDETypeVSCodeWeb || ideService.Type() == enum.IDETypeVSCode {
		if devcontainerConfig.Customizations.ExtractVSCodeSpec() != nil {
			args[gitspaceTypes.VSCodeCustomizationArg] = *devcontainerConfig.Customizations.ExtractVSCodeSpec()
		}
	}
	return args
}
