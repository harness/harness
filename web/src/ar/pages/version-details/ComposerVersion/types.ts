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

export enum ComposerArtifactDetailsTabEnum {
  ReadMe = 'readme',
  Files = 'files',
  Dependencies = 'dependencies'
}
export interface ComposerVersionDetailsQueryParams extends VersionDetailsQueryParams {
  detailsTab: ComposerArtifactDetailsTabEnum
}

export type LocalComposerArtifactDetailConfig = {
  metadata?: {
    name?: string
    version?: string
    description?: string
    size?: number
    readme?: string
    license?: string
    require?: Record<string, string>
    'require-dev'?: Record<string, string>
  }
}

export type ComposerArtifactDetails = ArtifactDetail & LocalComposerArtifactDetailConfig
