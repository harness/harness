/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useMemo } from 'react'
import { Checkbox, PaginationProps, TableV2 } from '@harnessio/uicore'
import type { CellProps, Column } from 'react-table'
import css from './ResourceHandlerTable.module.scss'

interface ResourceHandlerTableProps<T extends ResourceHandlerTableData> {
  data: T[]
  columns: Column<T>[]
  selectedData?: string[]
  disabledRows?: string[]
  pagination?: PaginationProps
  onSelectChange: (items: string[]) => void
  hideHeaders?: boolean
  minimal?: boolean
}

export interface ResourceHandlerTableData {
  id: number
  path: string
}

const ResourceHandlerTable = <T extends ResourceHandlerTableData>(
  props: ResourceHandlerTableProps<T>
): React.ReactElement => {
  const {
    data,
    pagination,
    columns,
    onSelectChange,
    selectedData = [],
    disabledRows = [],
    hideHeaders = false,
    minimal = false
  } = props

  const disabledIds = useMemo(() => disabledRows.map(d => Number(d.split('/').at(-1))), [disabledRows])
  const selectedIds = useMemo(() => selectedData.map(d => Number(d.split('/').at(-1))), [selectedData])

  const handleSelectChange = (isSelect: boolean, item: string): void => {
    onSelectChange(isSelect ? [...selectedData, item] : selectedData.filter(d => d !== item))
  }

  const resourceHandlerTableColumns: Column<T>[] = useMemo(
    () => [
      {
        id: 'enabled',
        accessor: 'id',
        width: '5%',
        disableSortBy: true,
        Cell: ({ row }: CellProps<T>) => {
          return (
            <Checkbox
              key={row.original.id}
              className={css.checkBox}
              disabled={disabledIds.includes(row.original.id)}
              defaultChecked={selectedIds.includes(row.original.id)}
              onChange={(event: React.FormEvent<HTMLInputElement>) => {
                handleSelectChange(event.currentTarget.checked, [row.original.path, row.original.id].join(' '))
              }}
            />
          )
        }
      },
      ...columns
    ],
    [selectedIds, disabledIds]
  )
  return (
    <TableV2<T>
      columns={resourceHandlerTableColumns}
      data={data}
      pagination={pagination}
      hideHeaders={hideHeaders}
      minimal={minimal}
      onRowClick={row => {
        handleSelectChange(!selectedIds.includes(row.id), [row.path, row.id].join('/'))
      }}
    />
  )
}

export default ResourceHandlerTable
