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

import React from 'react'
import classNames from 'classnames'
import type { Column } from 'react-table'
import { PaginationProps, TableV2 } from '@harnessio/uicore'
import type { ArtifactMetadata, ListArtifact } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useParentHooks } from '@ar/hooks'
import {
  ArtifactDeploymentsCell,
  ArtifactDownloadsCell,
  ArtifactListPullCommandCell,
  ArtifactListVulnerabilitiesCell,
  ArtifactNameCell,
  LatestArtifactCell
} from './ArtifactListTableCell'
import css from './ArtifactListTable.module.scss'

export interface ArtifactListColumnActions {
  refetchList?: () => void
}
export interface ArtifactListTableProps extends ArtifactListColumnActions {
  data: ListArtifact
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export default function ArtifactListTable(props: ArtifactListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy } = props
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

  const columns: Column<ArtifactMetadata>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: any) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: getString('artifactList.table.columns.artifactName'),
        accessor: 'name',
        Cell: ArtifactNameCell,
        serverSortProps: getServerSortProps('name')
      },
      {
        Header: getString('artifactList.table.columns.pullCommand'),
        accessor: 'pullCommand',
        Cell: ArtifactListPullCommandCell,
        disableSortBy: true
      },
      {
        Header: getString('artifactList.table.columns.downloads'),
        accessor: 'downloadsCount',
        Cell: ArtifactDownloadsCell,
        serverSortProps: getServerSortProps('downloadsCount')
      },
      {
        Header: getString('artifactList.table.columns.environments'),
        accessor: 'environments',
        Cell: ArtifactDeploymentsCell,
        disableSortBy: true
      },
      {
        Header: getString('artifactList.table.columns.sto'),
        accessor: 'sto',
        Cell: ArtifactListVulnerabilitiesCell,
        disableSortBy: true
      },
      {
        Header: getString('artifactList.table.columns.lastUpdated'),
        accessor: 'lastUpdated',
        Cell: LatestArtifactCell,
        serverSortProps: getServerSortProps('lastUpdated')
      }
    ].filter(Boolean) as unknown as Column<ArtifactMetadata>[]
  }, [currentOrder, currentSort, getString])

  return (
    <TableV2<ArtifactMetadata>
      className={classNames(css.table)}
      columns={columns}
      data={artifacts}
      pagination={paginationProps}
      sortable
      getRowClassName={() => css.tableRow}
    />
  )
}
