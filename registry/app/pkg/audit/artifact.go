//  Copyright 2023 Harness, Inc.
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

package audit

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/pkg"

	"github.com/rs/zerolog/log"
)

const (
	// Audit metadata keys.
	AuditKeyResourceName = "resourceName"
	AuditKeyArtifactUUID = "artifactId"
	AuditKeyImageUUID    = "imageUuid"
)

// LogArtifactPush logs audit trail for artifact push/upload operations.
// This is a centralized audit utility that can be called from any package type handler.
func LogArtifactPush(
	ctx context.Context,
	auditService audit.Service,
	spaceFinder refcache.SpaceFinder,
	info pkg.ArtifactInfo,
	version string,
	imageUUID string,
	artifactUUID string,
) {
	session, ok := request.AuthSessionFrom(ctx)
	if !ok {
		log.Ctx(ctx).Debug().Msg("no auth session for audit log")
		return
	}

	parentSpace, err := spaceFinder.FindByID(ctx, info.ParentID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to get parent space for audit log")
		return
	}

	packageName := info.Image
	artifactIdentifier := fmt.Sprintf("%s:%s", packageName, version)
	if version == "" {
		artifactIdentifier = packageName
	}

	// Operational metadata
	auditData := []audit.Option{
		audit.WithData(
			AuditKeyImageUUID, imageUUID,
		),
	}

	err = auditService.Log(
		ctx,
		session.Principal,
		audit.NewResource(
			audit.ResourceTypeRegistryArtifact,
			artifactIdentifier,
			AuditKeyResourceName, artifactIdentifier,
			AuditKeyArtifactUUID, artifactUUID,
		),
		audit.ActionUploaded,
		parentSpace.Path,
		auditData...,
	)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf(
			"failed to insert audit log for upload artifact operation: %s",
			artifactIdentifier,
		)
	}
}
