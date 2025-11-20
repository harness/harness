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

import type { IconName } from '@harnessio/icons'
import type { Scanner } from '@harnessio/react-har-service-client'

import { FeatureFlags } from '@ar/MFEAppTypes'
import type { StringKeys } from '@ar/frameworks/strings'
import { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'
import {
  CardSelectOption,
  Parent,
  RepositoryConfigType,
  RepositoryPackageType,
  RepositoryVisibility,
  Scanners
} from '@ar/common/types'

export const POLICY_TYPE = 'securityTests'
export const POLICY_ACTION = 'onstep'

export enum RepositoryDetailsTab {
  PACKAGES = 'packages',
  DATASETS = 'datasets',
  MODELS = 'models',
  CONFIGURATION = 'configuration',
  METADATA = 'metadata',
  WEBHOOKS = 'webhooks'
}

export enum LocalArtifactType {
  ARTIFACTS = 'artifacts',
  DATASET = 'dataset',
  MODEL = 'model'
}

export interface ScannerConfigSpec {
  icon: IconName
  label: string
  value: Scanner['name']
  tooltipId?: string
}

export const ContainerScannerConfig: Record<Scanners, ScannerConfigSpec> = {
  AQUA_TRIVY: {
    icon: 'AquaTrivy',
    label: 'AquaTrivy',
    value: 'AQUA_TRIVY'
  },
  GRYPE: {
    icon: 'anchore-grype',
    label: 'Grype',
    value: 'GRYPE'
  }
}

interface RepositoryDetailsTabSpec {
  label: StringKeys
  value: RepositoryDetailsTab
  artifactType?: LocalArtifactType
  packageType?: RepositoryPackageType
  type?: RepositoryConfigType
  mode?: RepositoryListViewTypeEnum
  featureFlag?: FeatureFlags
  isSupportedInPublicView?: boolean
  supportActions?: boolean
  parent?: Parent
}

export const RepositoryDetailsTabs: RepositoryDetailsTabSpec[] = [
  {
    label: 'repositoryDetails.tabs.packages',
    value: RepositoryDetailsTab.PACKAGES,
    mode: RepositoryListViewTypeEnum.LIST,
    artifactType: LocalArtifactType.ARTIFACTS,
    isSupportedInPublicView: true
  },
  {
    label: 'repositoryDetails.tabs.datasets',
    value: RepositoryDetailsTab.DATASETS,
    mode: RepositoryListViewTypeEnum.LIST,
    artifactType: LocalArtifactType.DATASET,
    isSupportedInPublicView: true
  },
  {
    label: 'repositoryDetails.tabs.models',
    value: RepositoryDetailsTab.MODELS,
    mode: RepositoryListViewTypeEnum.LIST,
    artifactType: LocalArtifactType.MODEL,
    isSupportedInPublicView: true
  },
  {
    label: 'repositoryDetails.tabs.configuration',
    value: RepositoryDetailsTab.CONFIGURATION,
    isSupportedInPublicView: false,
    supportActions: true
  },
  {
    label: 'repositoryDetails.tabs.webhooks',
    value: RepositoryDetailsTab.WEBHOOKS,
    featureFlag: FeatureFlags.HAR_TRIGGERS,
    type: RepositoryConfigType.VIRTUAL,
    isSupportedInPublicView: false
  },
  {
    label: 'metadata',
    value: RepositoryDetailsTab.METADATA,
    supportActions: true,
    featureFlag: FeatureFlags.HAR_CUSTOM_METADATA_ENABLED,
    parent: Parent.Enterprise
  }
]

export type RepositoryVisibilityOptionType = CardSelectOption<RepositoryVisibility>

export const RepositoryVisibilityOptions: Record<RepositoryVisibility, RepositoryVisibilityOptionType> = {
  [RepositoryVisibility.PUBLIC]: {
    label: 'repositoryDetails.repositoryForm.visibility.public',
    description: 'repositoryDetails.repositoryForm.visibility.publicDescription',
    value: RepositoryVisibility.PUBLIC
  },
  [RepositoryVisibility.PRIVATE]: {
    label: 'repositoryDetails.repositoryForm.visibility.private',
    description: 'repositoryDetails.repositoryForm.visibility.privateDescription',
    value: RepositoryVisibility.PRIVATE
  }
}
