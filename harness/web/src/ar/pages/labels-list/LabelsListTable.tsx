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
import classNames from 'classnames'
import type { Column, Row } from 'react-table'
import { Container, PaginationProps, TableV2 } from '@harnessio/uicore'

import type { TypesLabel } from 'services/code'

import { useParentHooks } from '@ar/hooks'
import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import { handleToggleExpandableRow } from '@ar/components/TableCells/utils'

import type { PaginationResponse } from './types'
import {
  LabelActionsCell,
  LabelAssociationsCell,
  LabelCreatedInCell,
  LabelDescriptionCell,
  LabelNameCell,
  ToggleAccordionCell
} from './components/TableCells/TableCells'
import LabelValuesList from './components/LabelValuesList/LabelValuesList'

import css from './LabelsListPage.module.scss'

interface LabelsListTableProps {
  labels: TypesLabel[]
  pagination: PaginationResponse
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  minimal?: boolean
  reload: () => void
}
function LabelsListTable(props: LabelsListTableProps) {
  const { labels, pagination, gotoPage, onPageSizeChange, minimal = true, reload } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const [expandedRows, setExpandedRows] = React.useState<Set<string>>(new Set())

  const { totalItems, totalPages, pageSize, page } = pagination
  const paginationProps = useDefaultPaginationProps({
    itemCount: totalItems,
    pageSize,
    pageCount: totalPages,
    pageIndex: page,
    gotoPage,
    onPageSizeChange
  })

  const getRowId = (rowData: TypesLabel) => {
    return `${rowData.key}-${rowData.id}`
  }

  const onToggleRow = useCallback((rowData: TypesLabel): void => {
    if (!rowData.value_count) return
    const value = getRowId(rowData)
    setExpandedRows(handleToggleExpandableRow(value))
  }, [])

  const columns: Column<TypesLabel>[] = React.useMemo(() => {
    return [
      {
        Header: '',
        accessor: 'select',
        id: 'rowSelectOrExpander',
        Cell: ToggleAccordionCell,
        disableSortBy: true,
        expandedRows,
        setExpandedRows,
        getRowId
      },
      {
        Header: getString('labelsList.table.columns.name'),
        accessor: 'key',
        Cell: LabelNameCell,
        disableSortBy: true
      },
      {
        Header: getString('labelsList.table.columns.scope'),
        accessor: 'scope',
        Cell: LabelCreatedInCell,
        disableSortBy: true
      },
      {
        Header: getString('labelsList.table.columns.description'),
        accessor: 'description',
        Cell: LabelDescriptionCell,
        disableSortBy: true
      },
      {
        Header: getString('labelsList.table.columns.associations'),
        accessor: 'associations',
        Cell: LabelAssociationsCell,
        disableSortBy: true
      },
      {
        Header: '',
        accessor: 'menu',
        Cell: LabelActionsCell,
        disableSortBy: true,
        reload: reload
      }
    ].filter(Boolean) as unknown as Column<TypesLabel>[]
  }, [getString, expandedRows, setExpandedRows, reload])

  const renderRowSubComponent = useCallback(
    ({ row }: { row: Row<TypesLabel> }) => (
      <Container className={css.tableRowSubComponent} onClick={killEvent}>
        <LabelValuesList data={row.original} />
      </Container>
    ),
    []
  )

  return (
    <TableV2
      className={classNames(css.table)}
      columns={columns}
      data={labels}
      pagination={paginationProps}
      sortable={false}
      getRowClassName={row => classNames(css.tableRow, { [css.activeRow]: expandedRows.has(getRowId(row.original)) })}
      onRowClick={onToggleRow}
      autoResetExpanded={false}
      renderRowSubComponent={renderRowSubComponent}
      minimal={minimal}
    />
  )
}

export default LabelsListTable
