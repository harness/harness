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

import React, { useCallback, useContext } from 'react'
import type { Column } from 'react-table'
import { PaginationProps, TableV2 } from '@harnessio/uicore'
import type { ListVersion, VersionMetadata } from '@harnessio/react-har-service-client'

import { useAppStore, useFeatureFlags, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { SoftDeleteFilterEnum } from '@ar/constants'
import type { RepositoryConfigType } from '@ar/common/types'
import type { SortByType } from '@ar/frameworks/Version/Version'
import { RepositoryProviderContext } from '@ar/pages/repository-details/context/RepositoryProvider'

import { getVersionListTableCellConfigs } from './utils'
import type { IVersionListTableColumnConfigType, VersionListColumnEnum } from './types'

import css from './VersionListTable.module.scss'

export interface ArtifactVersionListColumnActions {
  refetchList?: () => void
}
export interface CommonVersionListTableProps extends ArtifactVersionListColumnActions {
  data: ListVersion
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: SortByType) => void
  sortBy: SortByType
  minimal?: boolean
  columnConfigs: Partial<Record<VersionListColumnEnum, Partial<IVersionListTableColumnConfigType>>>
  softDeleteFilter?: SoftDeleteFilterEnum
}

function VersionListTable(props: CommonVersionListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy, columnConfigs, softDeleteFilter } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const featureFlags = useFeatureFlags()
  const { parent } = useAppStore()
  const { data: repositoryData } = useContext(RepositoryProviderContext)
  const { config } = repositoryData || {}
  const { type } = config || {}

  const { artifacts = [], itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data || {}
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })
  const [currentSort, currentOrder] = sortBy

  const getServerSortProps = useCallback(
    (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: any) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    },
    [currentOrder, currentSort]
  )

  const columns: Column<VersionMetadata>[] = React.useMemo(() => {
    return getVersionListTableCellConfigs(
      columnConfigs,
      getServerSortProps,
      getString,
      softDeleteFilter,
      parent,
      featureFlags,
      type as RepositoryConfigType
    ) as Column<VersionMetadata>[]
  }, [getServerSortProps, columnConfigs, getString, softDeleteFilter, parent, featureFlags, type])

  return (
    <TableV2<VersionMetadata>
      className={css.table}
      columns={columns}
      data={artifacts}
      pagination={paginationProps}
      sortable
    />
  )
}

export default VersionListTable
