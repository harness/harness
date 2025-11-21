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
import { type PaginationProps, TableV2 } from '@harnessio/uicore'
import type { FileDetail, ListFileDetail } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import {
  FileChecksumListCell,
  FileCreatedCell,
  FileDownloadCommandCell,
  FileNameCell,
  FileSizeCell
} from './ArtifactListTableCells'

import css from './ArtifactFileListTable.module.scss'

export interface FileListSortBy {
  sort: 'name' | 'size' | 'created'
}

interface ArtifactFileListTableProps {
  data: ListFileDetail
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
  className?: string
}

export default function ArtifactFileListTable(props: ArtifactFileListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy, className, minimal } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()

  const { files, itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data
  const paginationProps = useDefaultPaginationProps({
    showPagination: pageCount > 1,
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })

  const [currentSort, currentOrder] = sortBy

  const columns: Column<FileDetail>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: FileListSortBy) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: getString('versionDetails.artifactFiles.table.columns.name'),
        accessor: 'name',
        Cell: FileNameCell,
        serverSortProps: getServerSortProps('name'),
        width: ''
      },
      {
        Header: getString('versionDetails.artifactFiles.table.columns.size'),
        accessor: 'size',
        Cell: FileSizeCell,
        serverSortProps: getServerSortProps('size')
      },
      {
        Header: getString('versionDetails.artifactFiles.table.columns.checksum'),
        accessor: 'checksums',
        Cell: FileChecksumListCell,
        disableSortBy: true
      },
      {
        Header: getString('versionDetails.artifactFiles.table.columns.downloadCommand'),
        accessor: 'downloadCommand',
        Cell: FileDownloadCommandCell,
        disableSortBy: true
      },
      {
        Header: getString('versionDetails.artifactFiles.table.columns.created'),
        accessor: 'createdAt',
        Cell: FileCreatedCell,
        serverSortProps: getServerSortProps('createdAt')
      }
    ].filter(Boolean) as unknown as Column<FileDetail>[]
  }, [currentOrder, currentSort, getString])

  return (
    <TableV2
      className={classNames(css.table, className, {
        [css.minimal]: minimal
      })}
      columns={columns}
      data={files as FileDetail[]}
      pagination={paginationProps}
      sortable
      minimal={minimal}
    />
  )
}
