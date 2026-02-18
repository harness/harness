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
import type { ArtifactScanV3, ListArtifactScanResponseV3Response } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import {
  DependencyAndVersionCell,
  LastEvaluatedAtCell,
  RegistryNameCell,
  StatusCell,
  ViolationActionsCell
} from './components/TableCells/TableCells'
import css from './ViolationsListPage.module.scss'

export interface ViolationsListTableProps {
  data: ListArtifactScanResponseV3Response
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export default function ViolationsListTable(props: ViolationsListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange } = props

  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()

  const { data: scans = [], itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data || {}
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })

  const columns: Column<ArtifactScanV3>[] = React.useMemo(() => {
    return [
      {
        Header: getString('violationsList.table.columns.package'),
        accessor: 'packageName',
        Cell: DependencyAndVersionCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('violationsList.table.columns.registry'),
        accessor: 'registryName',
        Cell: RegistryNameCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('violationsList.table.columns.status'),
        accessor: 'scanStatus',
        Cell: StatusCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('violationsList.table.columns.lastEvaluatedAt'),
        accessor: 'lastEvaluatedAt',
        Cell: LastEvaluatedAtCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: '',
        accessor: 'action',
        Cell: ViolationActionsCell,
        disableSortBy: true,
        width: '100%'
      }
    ].filter(Boolean) as unknown as Column<ArtifactScanV3>[]
  }, [getString])

  return (
    <TableV2<ArtifactScanV3>
      className={classNames(css.table)}
      columns={columns}
      data={scans}
      pagination={paginationProps}
      sortable={false}
    />
  )
}
