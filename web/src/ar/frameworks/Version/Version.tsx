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
import type { ListArtifactVersion } from '@harnessio/react-har-service-client'
import type { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import type { RepositoryPackageType } from '@ar/common/types'

export interface VersionDetailsHeaderProps<T> {
  data: T
}

export interface VersionDetailsTabProps {
  tab: VersionDetailsTab
}

export interface VersionListTableProps {
  data: ListArtifactVersion
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export abstract class VersionStep<T> {
  protected abstract packageType: RepositoryPackageType
  protected abstract allowedVersionDetailsTabs: VersionDetailsTab[]

  getPackageType(): string {
    return this.packageType
  }

  getAllowedVersionDetailsTab(): VersionDetailsTab[] {
    return this.allowedVersionDetailsTabs
  }

  abstract renderVersionListTable(props: VersionListTableProps): JSX.Element

  abstract renderVersionDetailsHeader(props: VersionDetailsHeaderProps<T>): JSX.Element

  abstract renderVersionDetailsTab(props: VersionDetailsTabProps): JSX.Element
}
