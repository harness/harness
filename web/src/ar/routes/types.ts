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

import type { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'

export interface RepositoryDetailsPathParams {
  repositoryIdentifier: string
  tab?: RepositoryDetailsTab
}

export interface ArtifactDetailsPathParams extends RepositoryDetailsPathParams {
  artifactIdentifier: string
}

export interface VersionDetailsPathParams extends ArtifactDetailsPathParams {
  versionIdentifier: string
}

export interface VersionDetailsTabPathParams extends VersionDetailsPathParams {
  versionTab: string
  pipelineIdentifier?: string
  executionIdentifier?: string
  artifactId?: string
  sourceId?: string
}
