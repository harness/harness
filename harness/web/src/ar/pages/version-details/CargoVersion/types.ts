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

export enum CargoArtifactDetailsTabEnum {
  ReadMe = 'readme',
  Files = 'files',
  Dependencies = 'dependencies'
}
export interface CargoVersionDetailsQueryParams extends VersionDetailsQueryParams {
  detailsTab: CargoArtifactDetailsTabEnum
}

export type CargoArtifactDependency = {
  default_features: boolean
  kind: 'dev' | 'build' | 'normal'
  name: string
  version_req: string
}

export type LocalCargoArtifactDetailConfig = {
  metadata?: {
    name: string
    vers: string
    size: number
    yanked: boolean
    readme: string
    description: string
    license: string
    repository: string
    documentation: string
    homepage: string
    deps: CargoArtifactDependency[]
  }
}

export type CargoArtifactDetails = ArtifactDetail & LocalCargoArtifactDetailConfig
