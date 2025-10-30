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

import type { ArtifactSummary, RegistryArtifactMetadata } from '@harnessio/react-har-service-client'
import type { PageType } from '@ar/common/types'

export enum ArtifactActionsEnum {
  Delete = 'delete',
  SetupClient = 'setupClient'
}

export interface ArtifactActionProps {
  data: ArtifactSummary | RegistryArtifactMetadata
  artifactKey: string
  repoKey: string
  pageType: PageType
  readonly?: boolean
  onClose?: () => void
  allowedActions?: ArtifactActionsEnum[]
}
