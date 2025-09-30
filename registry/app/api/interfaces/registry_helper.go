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

package interfaces

import (
	"context"

	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/types"
)

type RegistryHelper interface {
	// DeleteVersion deletes the version
	DeleteVersion(ctx context.Context,
		regInfo *types.RegistryRequestBaseInfo,
		imageInfo *types.Image,
		artifactName string,
		versionName string) error

	// ReportDeleteVersionEvent reports the delete version event
	ReportDeleteVersionEvent(
		ctx context.Context,
		payload *registryevents.ArtifactDeletedPayload,
	)

	// ReportBuildPackageIndexEvent reports the build package index event
	ReportBuildPackageIndexEvent(ctx context.Context, registryID int64, artifactName string)

	// ReportBuildRegistryIndexEvent reports the build registry index event
	ReportBuildRegistryIndexEvent(ctx context.Context, registryID int64, sources []types.SourceRef)
}
