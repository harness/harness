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

import type { PaginationProps } from '@harnessio/uicore'
import type {
  ArtifactSummary,
  ArtifactVersionMetadata,
  ArtifactVersionSummary,
  ListVersion,
  RegistryArtifactMetadata,
  VersionMetadata
} from '@harnessio/react-har-service-client'
import type { PackageMetadata } from '@harnessio/react-har-service-client'

import type { PageType, Parent, RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import { ArtifactDetailsTab } from '@ar/pages/artifact-details/constants'
import type { SoftDeleteFilterEnum } from '@ar/constants'

export interface VersionDetailsHeaderProps<T> {
  data: T
}

export interface VersionDetailsTabProps {
  tab: VersionDetailsTab
}

export type SortByType = [string, 'ASC' | 'DESC']

export interface VersionListTableProps {
  data: ListVersion
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: SortByType) => void
  sortBy: SortByType
  minimal?: boolean
  parent: Parent
  softDeleteFilter?: SoftDeleteFilterEnum
}

export interface ArtifactActionProps {
  data: PackageMetadata | ArtifactSummary
  pageType: PageType
  repoKey: string
  artifactKey: string
  readonly?: boolean
  onClose?: () => void
}

export interface VersionActionProps {
  data: VersionMetadata | ArtifactVersionSummary
  pageType: PageType
  repoKey: string
  artifactKey: string
  versionKey: string
  digest?: string
  digestCount?: number
  readonly?: boolean
  onClose?: () => void
  repoType?: RepositoryConfigType
}

export interface ArtifactRowSubComponentProps {
  data: VersionMetadata
}

export interface ArtifactTreeNodeViewProps {
  data: RegistryArtifactMetadata
}

export interface VersionTreeNodeViewProps {
  data: ArtifactVersionMetadata
}

export interface VersionTreeNodeDetailsProps {
  data: ArtifactVersionSummary
}

export abstract class VersionStep<T> {
  protected abstract packageType: RepositoryPackageType
  protected abstract allowedVersionDetailsTabs: VersionDetailsTab[]
  protected abstract hasArtifactRowSubComponent: boolean
  supportedArtifactTabs?: ArtifactDetailsTab[]

  getPackageType(): string {
    return this.packageType
  }

  getAllowedVersionDetailsTab(): VersionDetailsTab[] {
    return this.allowedVersionDetailsTabs
  }

  getHasArtifactRowSubComponent(): boolean {
    return this.hasArtifactRowSubComponent
  }

  getSupportedArtifactTabs(): ArtifactDetailsTab[] {
    return this.supportedArtifactTabs ?? [ArtifactDetailsTab.VERSIONS, ArtifactDetailsTab.METADATA]
  }

  abstract renderVersionListTable(props: VersionListTableProps): JSX.Element

  abstract renderVersionDetailsHeader(props: VersionDetailsHeaderProps<T>): JSX.Element

  abstract renderVersionDetailsTab(props: VersionDetailsTabProps): JSX.Element

  abstract renderArtifactActions(props: ArtifactActionProps): JSX.Element

  abstract renderVersionActions(props: VersionActionProps): JSX.Element

  abstract renderArtifactRowSubComponent(props: ArtifactRowSubComponentProps): JSX.Element

  abstract renderArtifactTreeNodeView(props: ArtifactTreeNodeViewProps): JSX.Element

  abstract renderArtifactTreeNodeDetails(): JSX.Element

  abstract renderVersionTreeNodeView(props: VersionTreeNodeViewProps): JSX.Element

  abstract renderVersionTreeNodeDetails(props: VersionTreeNodeDetailsProps): JSX.Element
}
