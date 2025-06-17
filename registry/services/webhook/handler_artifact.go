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

package webhook

import (
	"context"
	"fmt"

	gitnesswebhook "github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ArtifactEventPayload describes the payload of Artifact related webhook triggers.
type ArtifactEventPayload struct {
	Trigger      enum.WebhookTrigger          `json:"trigger"`
	Registry     RegistryInfo                 `json:"registry"`
	Principal    gitnesswebhook.PrincipalInfo `json:"principal"`
	ArtifactInfo *registryevents.ArtifactInfo `json:"artifact_info"`
}

type RegistryInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// handleEventArtifactCreated handles branch created events
// and triggers branch created webhooks for the source repo.
func (s *Service) handleEventArtifactCreated(
	ctx context.Context,
	event *events.Event[*registryevents.ArtifactCreatedPayload],
) error {
	return s.triggerForEventWithArtifact(ctx, enum.WebhookTriggerArtifactCreated,
		event.ID, event.Payload.PrincipalID, event.Payload.RegistryID,
		func(
			principal *types.Principal,
			registry *registrytypes.Registry,
		) (any, error) {
			space, err := s.spaceStore.Find(ctx, registry.ParentID)
			if err != nil {
				return nil, err
			}
			return &ArtifactEventPayload{
				Trigger: enum.WebhookTriggerArtifactCreated,
				Registry: RegistryInfo{
					ID:          registry.ID,
					Name:        registry.Name,
					Description: registry.Description,
					URL:         s.urlProvider.GenerateUIRegistryURL(ctx, space.Path, registry.Name),
				},
				Principal: gitnesswebhook.PrincipalInfo{
					ID:          principal.ID,
					UID:         principal.UID,
					DisplayName: principal.DisplayName,
					Email:       principal.Email,
					Type:        principal.Type,
					Created:     principal.Created,
					Updated:     principal.Updated,
				},
				ArtifactInfo: getArtifactInfo(event.Payload.Artifact),
			}, nil
		})
}

// handleEventArtifactDeleted handles branch deleted events
// and triggers branch deleted webhooks for the source repo.
func (s *Service) handleEventArtifactDeleted(
	ctx context.Context,
	event *events.Event[*registryevents.ArtifactDeletedPayload],
) error {
	return s.triggerForEventWithArtifact(ctx, enum.WebhookTriggerArtifactDeleted,
		event.ID, event.Payload.PrincipalID, event.Payload.RegistryID,
		func(
			principal *types.Principal,
			registry *registrytypes.Registry,
		) (any, error) {
			space, err := s.spaceStore.Find(ctx, registry.ParentID)
			if err != nil {
				return nil, err
			}
			return &ArtifactEventPayload{
				Trigger: enum.WebhookTriggerArtifactDeleted,
				Registry: RegistryInfo{
					ID:          registry.ID,
					Name:        registry.Name,
					Description: registry.Description,
					URL:         s.urlProvider.GenerateUIRegistryURL(ctx, space.Path, registry.Name),
				},
				Principal: gitnesswebhook.PrincipalInfo{
					ID:          principal.ID,
					UID:         principal.UID,
					DisplayName: principal.DisplayName,
					Email:       principal.Email,
					Type:        principal.Type,
					Created:     principal.Created,
					Updated:     principal.Updated,
				},
				ArtifactInfo: getArtifactInfo(event.Payload.Artifact),
			}, nil
		})
}

func getArtifactInfo(eventArtifact registryevents.Artifact) *registryevents.ArtifactInfo {
	artifactInfo := registryevents.ArtifactInfo{}
	if dockerArtifact, ok := eventArtifact.(*registryevents.DockerArtifact); ok {
		artifactInfo.Type = artifact.PackageTypeDOCKER
		artifactInfo.Name = dockerArtifact.Name
		artifactInfo.Version = dockerArtifact.Tag
		artifactInfo.Artifact = &dockerArtifact
	} else if helmArtifact, ok := eventArtifact.(*registryevents.HelmArtifact); ok {
		artifactInfo.Type = artifact.PackageTypeHELM
		artifactInfo.Name = helmArtifact.Name
		artifactInfo.Version = helmArtifact.Tag
		artifactInfo.Artifact = &helmArtifact
	}
	return &artifactInfo
}

// triggerForEventWithArtifact triggers all webhooks for the given registry and triggerType
// using the eventID to generate a deterministic triggerID and using the output of bodyFn as payload.
// The method tries to find the registry and principal and provides both to the bodyFn to generate the body.
// NOTE: technically we could avoid this call if we send the data via the event (though then events will get big).
func (s *Service) triggerForEventWithArtifact(
	ctx context.Context,
	triggerType enum.WebhookTrigger,
	eventID string,
	principalID int64,
	registryID int64,
	createBodyFn func(*types.Principal, *registrytypes.Registry) (any, error),
) error {
	principal, err := s.WebhookExecutor.FindPrincipalForEvent(ctx, principalID)
	if err != nil {
		return err
	}
	registry, err := s.registryRepository.Get(ctx, registryID)
	if err != nil {
		return err
	}
	body, err := createBodyFn(principal, registry)
	if err != nil {
		return fmt.Errorf("body creation function failed: %w", err)
	}

	parents, err := s.getParentInfoRegistry(ctx, registry.ID, true)
	if err != nil {
		return fmt.Errorf("failed to get webhook parent info: %w", err)
	}

	return s.WebhookExecutor.TriggerForEvent(ctx, eventID, parents, triggerType, body)
}

func (s *Service) getParentInfoRegistry(
	ctx context.Context,
	registryID int64,
	inherited bool,
) ([]types.WebhookParentInfo, error) {
	var parents []types.WebhookParentInfo

	parents = append(parents, types.WebhookParentInfo{
		ID:   registryID,
		Type: enum.WebhookParentRegistry,
	})

	if inherited {
		registry, err := s.registryRepository.Get(ctx, registryID)
		if err != nil {
			return nil, fmt.Errorf("failed to get registry: %w", err)
		}

		ids, err := s.spaceStore.GetAncestorIDs(ctx, registry.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent space ids: %w", err)
		}

		for _, id := range ids {
			parents = append(parents, types.WebhookParentInfo{
				Type: enum.WebhookParentSpace,
				ID:   id,
			})
		}
	}

	return parents, nil
}
