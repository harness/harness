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

import type { ArtifactDetail } from '@harnessio/react-har-service-client'
import type { VersionDetailsQueryParams } from '../types'

export enum HuggingfaceArtifactDetailsTabEnum {
  ReadMe = 'readme',
  Files = 'files',
  Dependencies = 'dependencies'
}
export interface HuggingfaceVersionDetailsQueryParams extends VersionDetailsQueryParams {
  detailsTab: HuggingfaceArtifactDetailsTabEnum
}

export type HuggingfaceArtifactDependency = {
  name: string
  version: string
}

export type LocalHuggingfaceArtifactDetailConfig = {
  metadata?: {
    language: string
    projectUrl: string
    tags: string
    license: string
    readme: string
    dependencies: HuggingfaceArtifactDependency[]
  }
}

export type HuggingfaceArtifactDetails = ArtifactDetail & LocalHuggingfaceArtifactDetailConfig
