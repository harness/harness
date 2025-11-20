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

import React, { useCallback } from 'react'
import type { Column } from 'react-table'
import { PaginationProps, TableV2 } from '@harnessio/uicore'
import type { ArtifactMetadata, ListArtifact } from '@harnessio/react-har-service-v2-client'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { SortByType } from '@ar/frameworks/Version/Version'
import type { IVersionListTableColumnConfigType, VersionListColumnEnum } from './types'

import { getVersionListTableCellConfigs } from './utils'
import css from './VersionListTable.module.scss'

export interface ArtifactVersionListColumnActions {
  refetchList?: () => void
}
export interface CommonVersionListTableProps extends ArtifactVersionListColumnActions {
  data: ListArtifact
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: SortByType) => void
  sortBy: SortByType
  minimal?: boolean
  columnConfigs: Partial<Record<VersionListColumnEnum, Partial<IVersionListTableColumnConfigType>>>
}

function VersionListTable(props: CommonVersionListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy, columnConfigs } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()

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

  const columns: Column<ArtifactMetadata>[] = React.useMemo(() => {
    return getVersionListTableCellConfigs(columnConfigs, getServerSortProps, getString) as Column<ArtifactMetadata>[]
  }, [getServerSortProps, columnConfigs, getString])

  return (
    <TableV2<ArtifactMetadata>
      className={css.table}
      columns={columns}
      data={artifacts}
      pagination={paginationProps}
      sortable
    />
  )
}

export default VersionListTable
