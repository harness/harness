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

package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types"
)

// ResolveFeatures resolves all the downloaded features ie starting from the user-specified features, it checks
// which features need to be installed with which options. A feature in considered uniquely installable if its
// id or source and the options overrides are unique.
// Reference: https://containers.dev/implementors/features/#definition-feature-equality
func ResolveFeatures(
	userDefinedFeatures types.Features,
	downloadedFeatures map[string]*types.DownloadedFeature,
) (map[string]*types.ResolvedFeature, error) {
	featuresToBeResolved := make([]types.FeatureValue, 0)
	for _, featureValue := range userDefinedFeatures {
		featuresToBeResolved = append(featuresToBeResolved, *featureValue)
	}

	resolvedFeatures := make(map[string]*types.ResolvedFeature)
	for i := 0; i < len(featuresToBeResolved); i++ {
		currentFeature := featuresToBeResolved[i]

		digest, err := calculateDigest(currentFeature.Source, currentFeature.Options)
		if err != nil {
			return nil, fmt.Errorf("error calculating digest for %s: %w", currentFeature.Source, err)
		}

		if _, alreadyResolved := resolvedFeatures[digest]; alreadyResolved {
			continue
		}

		downloadedFeature := downloadedFeatures[currentFeature.Source]
		resolvedOptions, err := getResolvedOptions(downloadedFeature, currentFeature)
		if err != nil {
			return nil, err
		}

		resolvedFeature := types.ResolvedFeature{
			Digest:            digest,
			ResolvedOptions:   resolvedOptions,
			OverriddenOptions: currentFeature.Options, // used to calculate digest and sort features
			DownloadedFeature: downloadedFeature,
		}

		resolvedFeatures[digest] = &resolvedFeature
		if resolvedFeature.DownloadedFeature.DevcontainerFeatureConfig.DependsOn != nil &&
			len(*resolvedFeature.DownloadedFeature.DevcontainerFeatureConfig.DependsOn) > 0 {
			for _, featureValue := range *resolvedFeature.DownloadedFeature.DevcontainerFeatureConfig.DependsOn {
				featuresToBeResolved = append(featuresToBeResolved, *featureValue)
			}
		}
	}
	return resolvedFeatures, nil
}

func getResolvedOptions(
	downloadedFeature *types.DownloadedFeature,
	currentFeature types.FeatureValue,
) (map[string]string, error) {
	resolvedOptions := make(map[string]string)
	if downloadedFeature.DevcontainerFeatureConfig.Options != nil {
		for optionKey, optionDefinition := range *downloadedFeature.DevcontainerFeatureConfig.Options {
			var optionValue = optionDefinition.Default
			if userProvidedOptionValue, ok := currentFeature.Options[optionKey]; ok {
				optionValue = userProvidedOptionValue
			}
			stringValue, err := optionDefinition.ValidateValue(optionValue, optionKey, currentFeature.Source)
			if err != nil {
				return nil, err
			}
			resolvedOptions[optionKey] = stringValue
		}
	}
	return resolvedOptions, nil
}

// calculateDigest calculates a deterministic hash for a feature using its source and options overrides.
func calculateDigest(source string, optionsOverrides map[string]any) (string, error) {
	data := map[string]any{
		"options": optionsOverrides,
		"source":  source,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize data: %w", err)
	}

	hash := sha256.Sum256(jsonBytes)

	return hex.EncodeToString(hash[:]), nil
}
