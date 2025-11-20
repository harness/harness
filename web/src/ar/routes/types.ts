/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { ArtifactType } from '@harnessio/react-har-service-client'
import type { RepositoryPackageType } from '@ar/common/types'
import type { ManageRegistriesDetailsTab } from '@ar/pages/manage-registries/constants'
import type { LocalArtifactType, RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import type { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import type { WebhookDetailsTab } from '@ar/pages/webhook-details/constants'
import type { ArtifactDetailsTab } from '@ar/pages/artifact-details/constants'

export interface ManageRegistriesTabPathParams {
  tab: ManageRegistriesDetailsTab
}

export interface RepositoryDetailsPathParams {
  repositoryIdentifier: string
}

export interface RepositoryDetailsTabPathParams extends RepositoryDetailsPathParams {
  tab: RepositoryDetailsTab
}

export interface ArtifactDetailsPathParams extends RepositoryDetailsPathParams {
  artifactIdentifier: string
  artifactType: LocalArtifactType
}

export interface ArtifactDetailsTabPathParams extends ArtifactDetailsPathParams {
  tab: ArtifactDetailsTab
}

export interface VersionDetailsPathParams extends ArtifactDetailsPathParams {
  versionIdentifier: string
  tag?: string
  digest?: string
}

export interface VersionDetailsTabPathParams extends VersionDetailsPathParams {
  versionTab: string
  pipelineIdentifier?: string
  executionIdentifier?: string
  artifactId?: string
  sourceId?: string
  orgIdentifier?: string
  projectIdentifier?: string
}

export interface RedirectPageQueryParams {
  accountId?: string
  orgIdentifier?: string
  projectIdentifier?: string
  packageType: RepositoryPackageType
  registryId: string
  artifactId?: string
  versionId?: string
  versionDetailsTab?: VersionDetailsTab
  artifactType?: ArtifactType
  tag?: string
}

export interface RepositoryWebhookDetailsPathParams extends RepositoryDetailsPathParams {
  webhookIdentifier: string
}

export interface RepositoryWebhookDetailsTabPathParams extends RepositoryWebhookDetailsPathParams {
  tab: WebhookDetailsTab
}
