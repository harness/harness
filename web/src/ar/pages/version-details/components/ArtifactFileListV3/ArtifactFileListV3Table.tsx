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
import { Layout, TableV2 } from '@harnessio/uicore'
import type { ListFilesResponseBodyV3 } from '@harnessio/react-har-service-client'

import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import TablePaginationV3 from '@ar/components/TablePaginationV3/TablePaginationV3'

import { getFileListTableColumnsV3 } from './getArtifactFileListTableV3ColumnsV3'

import css from './ArtifactFileListV3.module.scss'

export interface FilesListV3TableProps {
  data: ListFilesResponseBodyV3
  sortBy: string[]
  onPageChange: (pageNumber: number) => void
  onPageSizeChange: (pageSize: number) => void
  setSortBy: (sortBy: string[]) => void
  repositoryIdentifier: string
  className?: string
}

export default function FilesListV3Table(props: FilesListV3TableProps): JSX.Element {
  const { data, setSortBy, sortBy, onPageChange, onPageSizeChange, repositoryIdentifier, className } = props
  const { getString } = useStrings()

  const rows = data.items ?? []
  const columns = React.useMemo(
    () => getFileListTableColumnsV3(getString, sortBy, setSortBy, repositoryIdentifier),
    [getString, sortBy, setSortBy, repositoryIdentifier]
  )

  const page = data.page ?? DEFAULT_PAGE_INDEX
  const size = data.size ?? 20
  const hasMore = data.hasMore ?? false

  return (
    <Layout.Vertical>
      <TableV2 className={classNames(css.table, className)} columns={columns} data={rows} sortable />
      <TablePaginationV3
        pageSize={size}
        page={page}
        hasMore={hasMore}
        gotoPage={onPageChange}
        onPageSizeChange={onPageSizeChange}
      />
    </Layout.Vertical>
  )
}
