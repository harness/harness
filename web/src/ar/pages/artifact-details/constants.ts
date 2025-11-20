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

import { FeatureFlags } from '@ar/MFEAppTypes'
import type { StringKeys } from '@ar/frameworks/strings'
import { Parent, type RepositoryConfigType, type RepositoryPackageType } from '@ar/common/types'
import type { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'
import type { LocalArtifactType } from '../repository-details/constants'

export enum ArtifactDetailsTab {
  VERSIONS = 'versions',
  METADATA = 'metadata'
}

interface ArtifactDetailsTabSpec {
  label: StringKeys
  value: ArtifactDetailsTab
  artifactType?: LocalArtifactType
  packageType?: RepositoryPackageType
  type?: RepositoryConfigType
  mode?: RepositoryListViewTypeEnum
  featureFlag?: FeatureFlags
  supportActions?: boolean
  parent?: Parent
}

export const ArtifactDetailsTabs: ArtifactDetailsTabSpec[] = [
  {
    label: 'artifactDetails.tabs.versions',
    value: ArtifactDetailsTab.VERSIONS
  },
  {
    label: 'metadata',
    value: ArtifactDetailsTab.METADATA,
    supportActions: true,
    featureFlag: FeatureFlags.HAR_CUSTOM_METADATA_ENABLED,
    parent: Parent.Enterprise
  }
]
