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
  RegistryArtifactDownloadsCell,
  RegistryArtifactLatestUpdatedCell,
  RegistryArtifactNameCell,
  RepositoryNameCell
} from './RegistryArtifactListTableCell'
import css from './RegistryArtifactListTable.module.scss'

export interface RegistryArtifactListColumnActions {
  refetchList?: () => void
}
export interface RegistryArtifactListTableProps extends RegistryArtifactListColumnActions {
  data: ListArtifact
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
  onClickLabel: (val: string) => void
}

export default function RegistryArtifactListTable(props: RegistryArtifactListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy, onClickLabel } = props
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
        Header: getString('artifactList.table.columns.name'),
        accessor: 'name',
        Cell: RegistryArtifactNameCell,
        serverSortProps: getServerSortProps('name'),
        onClickLabel
      },
      {
        Header: getString('artifactList.table.columns.repository'),
        accessor: 'registryIdentifier',
        Cell: RepositoryNameCell,
        serverSortProps: getServerSortProps('registryIdentifier')
      },
      {
        Header: getString('artifactList.table.columns.downloads'),
        accessor: 'downloadsCount',
        Cell: RegistryArtifactDownloadsCell,
        serverSortProps: getServerSortProps('downloadsCount')
      },
      {
        Header: getString('artifactList.table.columns.latestVersion'),
        accessor: 'latestVersion',
        Cell: RegistryArtifactLatestUpdatedCell,
        serverSortProps: getServerSortProps('latestVersion')
      }
    ].filter(Boolean) as unknown as Column<ArtifactMetadata>[]
  }, [currentOrder, currentSort, getString, onClickLabel])

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
