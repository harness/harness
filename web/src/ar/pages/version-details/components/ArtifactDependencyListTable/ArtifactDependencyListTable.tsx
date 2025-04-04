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
import type { Column } from 'react-table'
import { TableV2 } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import type { IDependencyItem, IDependencyList } from './types'
import { DependencyNameCell, DependencyVersionCell } from './ArtifactDependencyListTableCells'

interface ArtifactDependencyListTableProps {
  data: IDependencyList
  minimal?: boolean
  className?: string
}

export default function ArtifactDependencyListTable(props: ArtifactDependencyListTableProps): JSX.Element {
  const { data, className } = props
  const { getString } = useStrings()

  const columns: Column<IDependencyItem>[] = React.useMemo(() => {
    return [
      {
        Header: getString('versionDetails.dependencyList.table.columns.name'),
        accessor: 'name',
        Cell: DependencyNameCell,
        width: '100%'
      },
      {
        Header: getString('versionDetails.dependencyList.table.columns.version'),
        accessor: 'version',
        Cell: DependencyVersionCell,
        width: '20%'
      }
    ].filter(Boolean) as unknown as Column<IDependencyItem>[]
  }, [getString])

  return <TableV2 className={className} columns={columns} data={data} sortable={false} />
}
