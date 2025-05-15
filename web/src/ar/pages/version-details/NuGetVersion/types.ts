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

export enum NuGetArtifactDetailsTabEnum {
  ReadMe = 'readme',
  Files = 'files',
  Dependencies = 'dependencies'
}
export interface NuGetVersionDetailsQueryParams extends VersionDetailsQueryParams {
  detailsTab: NuGetArtifactDetailsTabEnum
}

interface IDependency {
  id: string
  version: string
}

export interface IDependencyGroup {
  dependencies?: IDependency[]
  targetFramework: string
}

export interface LocalNugetArtifactDetailConfig {
  metadata?: {
    metadata: {
      authors?: string
      description?: string
      id: string
      language?: string
      licenseUrl?: string
      repository?: {
        url: string
      }
      license?: {
        Text: string
        type: string
      }
      owners?: string
      projectUrl?: string
      tags?: string
      title?: string
      version: string
      readme?: string
      dependencies?: {
        dependencies?: IDependency[]
        groups?: IDependencyGroup[]
      }
    }
  }
}

export type NugetArtifactDetails = ArtifactDetail & LocalNugetArtifactDetailConfig
