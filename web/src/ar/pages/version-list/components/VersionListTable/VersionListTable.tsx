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
import type { ArtifactVersionMetadata, ListArtifactVersion } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
import { useAppStore, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import {
  PullCommandCell,
  VersionDeploymentsCell,
  VersionDigestsCell,
  VersionDownloadsCell,
  VersionNameCell,
  VersionPublishedAtCell,
  VersionSizeCell
} from './VersionListCell'

import css from './VersionListTable.module.scss'

export interface ArtifactVersionListColumnActions {
  refetchList?: () => void
}
export interface VersionListTableProps extends ArtifactVersionListColumnActions {
  data: ListArtifactVersion
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

function VersionListTable(props: VersionListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const { parent } = useAppStore()

  const { artifactVersions = [], itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data || {}
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })
  const [currentSort, currentOrder] = sortBy

  const columns: Column<ArtifactVersionMetadata>[] = React.useMemo(() => {
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
        Header: getString('versionList.table.columns.version'),
        accessor: 'name',
        Cell: VersionNameCell,
        serverSortProps: getServerSortProps('name')
      },
      {
        Header: getString('versionList.table.columns.deployments'),
        accessor: 'deployments',
        Cell: VersionDeploymentsCell,
        serverSortProps: getServerSortProps('deployments'),
        hidden: parent === Parent.OSS
      },
      {
        Header: getString('versionList.table.columns.size'),
        accessor: 'size',
        Cell: VersionSizeCell,
        serverSortProps: getServerSortProps('size')
      },
      {
        Header: getString('versionList.table.columns.digests'),
        accessor: 'digestCount',
        Cell: VersionDigestsCell,
        serverSortProps: getServerSortProps('digestCount')
      },
      {
        Header: getString('versionList.table.columns.downloads'),
        accessor: 'downloadsCount',
        Cell: VersionDownloadsCell,
        serverSortProps: getServerSortProps('downloadsCount')
      },
      {
        Header: getString('versionList.table.columns.publishedByAt'),
        accessor: 'lastModified',
        Cell: VersionPublishedAtCell,
        serverSortProps: getServerSortProps('lastModified')
      },
      {
        Header: getString('versionList.table.columns.pullCommand'),
        accessor: 'pullCommand',
        Cell: PullCommandCell,
        serverSortProps: getServerSortProps('pullCommand')
      }
    ]
      .filter(Boolean)
      .filter(each => !each.hidden) as unknown as Column<ArtifactVersionMetadata>[]
  }, [currentOrder, currentSort, getString])

  return (
    <TableV2<ArtifactVersionMetadata>
      className={classNames(css.table, {
        [css.ossTable]: parent === Parent.OSS,
        [css.enterpriseTable]: parent === Parent.Enterprise
      })}
      columns={columns}
      data={artifactVersions}
      pagination={paginationProps}
      sortable
      getRowClassName={() => css.tableRow}
    />
  )
}

export default VersionListTable
