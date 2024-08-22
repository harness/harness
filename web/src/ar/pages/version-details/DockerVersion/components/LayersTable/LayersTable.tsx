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

import React, { useMemo } from 'react'
import classNames from 'classnames'
import type { Column } from 'react-table'
import { TableV2 } from '@harnessio/uicore'
import type { DockerLayerEntry } from '@harnessio/react-har-service-client'

import { LayerActionCell, LayerCodeCell, LayerIndexCell, LayerSizeCell } from './LayersTableCells'
import css from './LayersTable.module.scss'

interface LayersTableProps {
  data: DockerLayerEntry[]
}

export default function LayersTable(props: LayersTableProps): JSX.Element {
  const { data } = props

  const columns: Column<any>[] = React.useMemo(() => {
    return [
      {
        accessor: 'id',
        Cell: LayerIndexCell
      },
      {
        accessor: 'code',
        Cell: LayerCodeCell
      },
      {
        accessor: 'size',
        Cell: LayerSizeCell
      },
      {
        accessor: 'action',
        Cell: LayerActionCell
      }
    ].filter(Boolean) as unknown as Column<any>[]
  }, [])

  const transformedData = useMemo(() => {
    if (Array.isArray(data)) {
      return data.map((each, index) => ({
        id: index + 1,
        code: each.command,
        size: '0 B'
      }))
    }
    return []
  }, [data])

  return (
    <TableV2<any>
      className={classNames(css.table)}
      columns={columns}
      data={transformedData}
      hideHeaders
      getRowClassName={() => css.tableRow}
    />
  )
}
