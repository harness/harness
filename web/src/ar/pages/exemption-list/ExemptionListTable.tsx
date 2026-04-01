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
import { TableV2 } from '@harnessio/uicore'
import type {
  FirewallExceptionResponseV3,
  ListFirewallExceptionsResponseBodyV3
} from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import TablePaginationV3 from '@ar/components/TablePaginationV3/TablePaginationV3'

import {
  ExemptionActionsCell,
  ExemptionUpdatedAtCell,
  ExpireAtCell,
  PackageNameCell,
  RepositoryNameCell,
  RequestedAtCell,
  StatusCell,
  VersionListCell
} from './components/TableCells/TableCells'

import css from './ExemptionListPage.module.scss'

export interface ExemptionListTableProps {
  data: ListFirewallExceptionsResponseBodyV3
  gotoPage: (pageNumber: number) => void
  onPageSizeChange: (size: number) => void
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export default function ExemptionListTable(props: ExemptionListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, setSortBy, sortBy } = props

  const { getString } = useStrings()
  const [currentSort, currentOrder] = sortBy

  const { items: exemptions = [], hasMore, page, size } = data || {}

  const columns: Column<FirewallExceptionResponseV3>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: { sort: string }) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: getString('exemptionList.table.columns.packageName'),
        accessor: 'packageName',
        Cell: PackageNameCell,
        serverSortProps: getServerSortProps('packageName'),
        width: '100%'
      },
      {
        Header: getString('exemptionList.table.columns.versions'),
        accessor: 'versionList',
        Cell: VersionListCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('exemptionList.table.columns.upstreamRegistry'),
        accessor: 'registryName',
        Cell: RepositoryNameCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('exemptionList.table.columns.status'),
        accessor: 'status',
        Cell: StatusCell,
        serverSortProps: getServerSortProps('status'),
        width: '100%'
      },
      {
        Header: getString('exemptionList.table.columns.requestedAt'),
        accessor: 'createdAt',
        Cell: RequestedAtCell,
        serverSortProps: getServerSortProps('createdAt'),
        width: '100%'
      },
      {
        Header: getString('exemptionList.table.columns.updatedAt'),
        accessor: 'updatedAt',
        Cell: ExemptionUpdatedAtCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('exemptionList.table.columns.expiresAt'),
        accessor: 'expirationAt',
        Cell: ExpireAtCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: '',
        accessor: 'exceptionId',
        Cell: ExemptionActionsCell,
        disableSortBy: true,
        width: '100%'
      }
    ].filter(Boolean) as unknown as Column<FirewallExceptionResponseV3>[]
  }, [getString, currentOrder, currentSort, getString])

  return (
    <>
      <TableV2<FirewallExceptionResponseV3>
        className={classNames(css.table)}
        columns={columns}
        data={exemptions}
        sortable
      />
      <TablePaginationV3
        pageSize={size}
        page={page}
        hasMore={hasMore}
        gotoPage={gotoPage}
        onPageSizeChange={onPageSizeChange}
      />
    </>
  )
}
