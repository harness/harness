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
import { Icon } from '@harnessio/icons'
import { Layout, Text } from '@harnessio/uicore'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance, UseExpandedRowProps } from 'react-table'

import type { TypesLabel } from 'services/code'
import { ColorName, getScopeData } from 'utils/Utils'

import { useAppStore } from '@ar/hooks'
import { Parent } from '@ar/common/types'
import { LabelTitle } from 'components/Label/Label'
import TableCells from '@ar/components/TableCells/TableCells'
import LabelActions from '../LabelActions/LabelActions'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<TypesLabel>>

export type LabelListExpandedColumnProps = {
  expandedRows: Set<string>
  setExpandedRows: React.Dispatch<React.SetStateAction<Set<string>>>
  getRowId: (rowData: TypesLabel) => string
}

export const ToggleAccordionCell: Renderer<{
  row: UseExpandedRowProps<TypesLabel> & Row<TypesLabel>
  column: ColumnInstance<TypesLabel> & LabelListExpandedColumnProps
}> = ({ row, column }) => {
  const { expandedRows, setExpandedRows, getRowId } = column
  const data = row.original
  if (!data.value_count) return <></>
  return (
    <TableCells.ToggleAccordionCell
      expandedRows={expandedRows}
      setExpandedRows={setExpandedRows}
      value={getRowId(data)}
      initialIsExpanded={row.isExpanded}
      getToggleRowExpandedProps={row.getToggleRowExpandedProps}
      onToggleRowExpanded={row.toggleRowExpanded}
    />
  )
}

export const LabelNameCell: CellType = ({ row }) => {
  return (
    <LabelTitle
      name={row.original?.key as string}
      value_count={row.original.value_count}
      label_color={row.original.color as ColorName}
      scope={row.original.scope}
    />
  )
}

export const LabelCreatedInCell: CellType = ({ row }) => {
  const { parent, scope } = useAppStore()
  const { scopeIcon, scopeId } = getScopeData(scope?.space as string, row.original.scope ?? 1, parent === Parent.OSS)

  return (
    <Layout.Horizontal spacing={'xsmall'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      <Icon size={16} name={scopeIcon} />
      <Text>{scopeId}</Text>
    </Layout.Horizontal>
  )
}

export const LabelDescriptionCell: CellType = ({ value }) => {
  return <TableCells.TextCell value={value} />
}
export const LabelAssociationsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="store-artifact-bundle" />
}

export type LabelListActionColumnProps = {
  reload: () => void
}

export const LabelActionsCell: Renderer<{
  row: UseExpandedRowProps<TypesLabel> & Row<TypesLabel>
  column: ColumnInstance<TypesLabel> & LabelListActionColumnProps
}> = ({ row, column }) => {
  const handleClose = (reload: boolean): void => {
    if (!reload) return
    column.reload()
  }
  return <LabelActions data={row.original} onClose={handleClose} />
}
