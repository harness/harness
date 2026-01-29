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

import React, { useCallback, useState } from 'react'
import classNames from 'classnames'
import type { Column, Row } from 'react-table'
import { Container, PaginationProps, TableV2 } from '@harnessio/uicore'
import type { ArtifactScan, ListArtifactScanResponseResponse } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import { handleToggleExpandableRow } from '@ar/components/TableCells/utils'

import {
  DependencyAndVersionCell,
  PolicySetName,
  PolicySetSpec,
  RegistryNameCell,
  StatusCell,
  ToggleAccordionCell,
  ViolationActionsCell
} from './components/TableCells/TableCells'
import css from './ViolationsListPage.module.scss'

export interface ViolationsListTableProps {
  data: ListArtifactScanResponseResponse
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export default function ViolationsListTable(props: ViolationsListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange } = props
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set())

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

  const getRowId = (rowData: ArtifactScan) => {
    return `${rowData.registryId}/${rowData.packageName}:${rowData.version}`
  }

  const onToggleRow = useCallback((rowData: ArtifactScan): void => {
    const value = getRowId(rowData)
    setExpandedRows(handleToggleExpandableRow(value))
  }, [])

  const columns: Column<ArtifactScan>[] = React.useMemo(() => {
    return [
      {
        Header: '',
        accessor: 'select',
        id: 'rowSelectOrExpander',
        Cell: ToggleAccordionCell,
        disableSortBy: true,
        expandedRows,
        setExpandedRows,
        getRowId,
        width: '10%'
      },
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
      }
    ].filter(Boolean) as unknown as Column<ArtifactScan>[]
  }, [getString, expandedRows, setExpandedRows])

  const subRowColumns: Column<PolicySetSpec>[] = React.useMemo(() => {
    return [
      {
        Header: getString('violationsList.table.columns.policySet'),
        accessor: 'name',
        Cell: PolicySetName,
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
    ].filter(Boolean) as unknown as Column<PolicySetSpec>[]
  }, [getString])

  const renderRowSubComponent = useCallback(
    ({ row }: { row: Row<ArtifactScan> }) => (
      <Container onClick={killEvent}>
        <TableV2<PolicySetSpec>
          className={classNames(css.table)}
          columns={subRowColumns}
          data={row.original.policySets.map(each => ({
            name: each.policySetName,
            identifier: each.policySetRef,
            scanId: row.original.id
          }))}
          sortable={false}
        />
      </Container>
    ),
    [subRowColumns]
  )

  return (
    <TableV2<ArtifactScan>
      className={classNames(css.table)}
      columns={columns}
      data={scans}
      pagination={paginationProps}
      sortable={false}
      renderRowSubComponent={renderRowSubComponent}
      getRowClassName={row => classNames(css.tableRow, { [css.activeRow]: expandedRows.has(getRowId(row.original)) })}
      onRowClick={onToggleRow}
      autoResetExpanded={false}
    />
  )
}
