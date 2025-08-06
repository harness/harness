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

import type { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'

export interface IGlobalFilters {
  repositoryTypes?: RepositoryPackageType[]
  configType?: RepositoryConfigType
  space?: string
}

export interface APIQueryParams {
  page: number
  size: number
  sort?: string
  space?: string
  searchTerm?: string
  packageTypes?: RepositoryPackageType[]
  configType?: RepositoryConfigType
  repositoryIdentifier?: string
  artifactIdentifier?: string
  versionIdentifier?: string
  digest?: string
}

export enum TreeNodeEntityEnum {
  REGISTRY = 'REGISTRY',
  ARTIFACT_TYPE = 'ARTIFACT_TYPE',
  ARTIFACT = 'ARTIFACT',
  VERSION = 'VERSION',
  DIGEST = 'DIGEST'
}
