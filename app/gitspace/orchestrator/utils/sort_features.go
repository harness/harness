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
	"fmt"
	"slices"

	"github.com/harness/gitness/types"
)

type node struct {
	Digest       string
	Dependencies map[string]bool
	Priority     int
	Feature      *types.ResolvedFeature
}

type featureWithPriority struct {
	Priority int
	Feature  *types.ResolvedFeature
}

// SortFeatures sorts the features topologically, using round priorities and feature installation order override
// if provided by the user. It also keeps track of hard dependencies and soft dependencies.
// Reference: https://containers.dev/implementors/features/#dependency-installation-order-algorithm.
func SortFeatures(
	featuresToBeInstalled map[string]*types.ResolvedFeature,
	overrideFeatureInstallOrder []string,
) ([]*types.ResolvedFeature, error) {
	sourcesWithoutTagsMappedToDigests := getSourcesWithoutTagsMappedToDigests(featuresToBeInstalled)

	adjacencyList, err := buildAdjacencyList(featuresToBeInstalled, sourcesWithoutTagsMappedToDigests,
		overrideFeatureInstallOrder)
	if err != nil {
		return nil, err
	}

	sortedFeatures, err := applyTopologicalSorting(adjacencyList)
	if err != nil {
		return nil, err
	}

	return sortedFeatures, nil
}

// getSourcesWithoutTagsMappedToDigests is used to map feature source (without tags) to their digests.
// Multiple features in the install set can have the same source but different options. All these features
// will be mapped to the same source and must be installed before any dependent features.
func getSourcesWithoutTagsMappedToDigests(featuresToBeInstalled map[string]*types.ResolvedFeature) map[string][]string {
	sourcesWithoutTagsMappedToDigests := map[string][]string{}
	for _, featureToBeInstalled := range featuresToBeInstalled {
		sourceWithoutTag := featureToBeInstalled.DownloadedFeature.SourceWithoutTag
		if _, initialized := sourcesWithoutTagsMappedToDigests[sourceWithoutTag]; !initialized {
			sourcesWithoutTagsMappedToDigests[sourceWithoutTag] = []string{}
		}
		sourcesWithoutTagsMappedToDigests[sourceWithoutTag] =
			append(sourcesWithoutTagsMappedToDigests[sourceWithoutTag], featureToBeInstalled.Digest)
	}
	return sourcesWithoutTagsMappedToDigests
}

func buildAdjacencyList(
	featuresToBeInstalled map[string]*types.ResolvedFeature,
	sourcesWithoutTagsMappedToDigests map[string][]string,
	overrideFeatureInstallOrder []string,
) ([]*node, error) {
	counter := 0
	adjacencyList := make([]*node, 0)
	for _, featureToBeInstalled := range featuresToBeInstalled {
		dependencies := map[string]bool{}

		err := populateHardDependencies(featureToBeInstalled, dependencies)
		if err != nil {
			return nil, err
		}

		populateSoftDependencies(sourcesWithoutTagsMappedToDigests, featureToBeInstalled, dependencies)

		// While the default priority is 0, it can be varied by the user through the overrideFeatureInstallOrder
		// in the devcontainer.json.
		// Reference: https://containers.dev/implementors/features/#overrideFeatureInstallOrder.
		priority := 0
		index := slices.Index(overrideFeatureInstallOrder, featureToBeInstalled.DownloadedFeature.SourceWithoutTag)
		if index > -1 {
			priority = len(overrideFeatureInstallOrder) - index
			counter++
		}

		graphNode := node{
			Digest:       featureToBeInstalled.Digest,
			Dependencies: dependencies,
			Priority:     priority,
			Feature:      featureToBeInstalled,
		}

		adjacencyList = append(adjacencyList, &graphNode)
	}

	// If any feature mentioned by the user in the overrideFeatureInstallOrder is not present in the install set,
	// fail the flow.
	difference := len(overrideFeatureInstallOrder) - counter
	if difference > 0 {
		return nil, fmt.Errorf("overrideFeatureInstallOrder contains %d extra features", difference)
	}

	return adjacencyList, nil
}

// populateSoftDependencies populates the digests of all the features whose source name is present in the installAfter
// property for the current feature ie which must be installed before the current feature can be installed.
// Any feature mentioned in the installAfter but not part of the install set is ignored.
func populateSoftDependencies(
	sourcesWithoutTagsMappedToDigests map[string][]string,
	featureToBeInstalled *types.ResolvedFeature,
	dependencies map[string]bool,
) {
	softDependencies := featureToBeInstalled.DownloadedFeature.DevcontainerFeatureConfig.InstallsAfter
	if len(softDependencies) > 0 {
		for _, softDependency := range softDependencies {
			if digests, ok := sourcesWithoutTagsMappedToDigests[softDependency]; ok {
				for _, digest := range digests {
					if _, alreadyAdded := dependencies[digest]; !alreadyAdded {
						dependencies[digest] = true
					}
				}
			}
		}
	}
}

// populateHardDependencies populates the digests of all the features which must be installed before the current
// feature can be installed.
func populateHardDependencies(featureToBeInstalled *types.ResolvedFeature, dependencies map[string]bool) error {
	hardDependencies := featureToBeInstalled.DownloadedFeature.DevcontainerFeatureConfig.DependsOn
	if hardDependencies != nil && len(*hardDependencies) > 0 {
		for _, hardDependency := range *hardDependencies {
			digest, err := calculateDigest(hardDependency.Source, hardDependency.Options)
			if err != nil {
				return fmt.Errorf("error calculating digest for %s: %w", hardDependency.Source, err)
			}
			dependencies[digest] = true
		}
	}
	return nil
}

func applyTopologicalSorting(
	adjacencyList []*node,
) ([]*types.ResolvedFeature, error) {
	sortedFeatures := make([]*types.ResolvedFeature, 0)
	for len(sortedFeatures) < len(adjacencyList) {
		maxPriority, eligibleFeatures := getFeaturesEligibleInThisRound(adjacencyList)

		if len(eligibleFeatures) == 0 {
			return nil, fmt.Errorf("features can not be sorted")
		}

		selectedFeatures := []*types.ResolvedFeature{}

		// only select those features which have the max priority, rest will be picked up in the next iteration.
		for _, eligibleFeature := range eligibleFeatures {
			if eligibleFeature.Priority == maxPriority {
				selectedFeatures = append(selectedFeatures, eligibleFeature.Feature)
			}
		}

		slices.SortStableFunc(selectedFeatures, types.CompareResolvedFeature)

		for _, selectedFeature := range selectedFeatures {
			sortedFeatures = append(sortedFeatures, selectedFeature)
			for _, vertex := range adjacencyList {
				delete(vertex.Dependencies, selectedFeature.Digest)
			}
		}
	}

	return sortedFeatures, nil
}

func getFeaturesEligibleInThisRound(adjacencyList []*node) (int, []featureWithPriority) {
	maxPriorityInRound := 0
	eligibleFeatures := []featureWithPriority{}
	for _, vertex := range adjacencyList {
		if len(vertex.Dependencies) == 0 {
			eligibleFeatures = append(eligibleFeatures, featureWithPriority{
				Priority: vertex.Priority,
				Feature:  vertex.Feature,
			})
			if maxPriorityInRound < vertex.Priority {
				maxPriorityInRound = vertex.Priority
			}
		}
	}
	return maxPriorityInRound, eligibleFeatures
}
