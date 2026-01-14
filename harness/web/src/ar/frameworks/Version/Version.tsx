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
  ArtifactMetadata,
  ArtifactSummary,
  ArtifactVersionMetadata,
  ArtifactVersionSummary,
  ListArtifactVersion,
  RegistryArtifactMetadata
} from '@harnessio/react-har-service-client'
import type { PageType, Parent, RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

export interface VersionDetailsHeaderProps<T> {
  data: T
}

export interface VersionDetailsTabProps {
  tab: VersionDetailsTab
}

export type SortByType = [string, 'ASC' | 'DESC']

export interface VersionListTableProps {
  data: ListArtifactVersion
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: SortByType) => void
  sortBy: SortByType
  minimal?: boolean
  parent: Parent
}

export interface ArtifactActionProps {
  data: RegistryArtifactMetadata | ArtifactSummary
  pageType: PageType
  repoKey: string
  artifactKey: string
  readonly?: boolean
  onClose?: () => void
}

export interface VersionActionProps {
  data: ArtifactVersionMetadata | ArtifactVersionSummary
  pageType: PageType
  repoKey: string
  artifactKey: string
  versionKey: string
  digest?: string
  digestCount?: number
  readonly?: boolean
  onClose?: () => void
}

export interface ArtifactRowSubComponentProps {
  data: ArtifactMetadata
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

  getPackageType(): string {
    return this.packageType
  }

  getAllowedVersionDetailsTab(): VersionDetailsTab[] {
    return this.allowedVersionDetailsTabs
  }

  getHasArtifactRowSubComponent(): boolean {
    return this.hasArtifactRowSubComponent
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
