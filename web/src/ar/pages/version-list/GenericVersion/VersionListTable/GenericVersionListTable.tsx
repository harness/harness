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
import type { Column } from 'react-table'
import classNames from 'classnames'
import type { ArtifactVersionMetadata, ListArtifactVersion } from '@harnessio/react-har-service-client'
import { PaginationProps, TableV2 } from '@harnessio/uicore'
import { Parent } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useParentHooks } from '@ar/hooks'
import type { SortByType } from '@ar/frameworks/Version/Version'
import {
  VersionDeploymentsCell,
  VersionDigestsCell,
  VersionNameCell,
  VersionPublishedAtCell
} from '../../components/VersionListTable/VersionListCell'

import css from '@ar/pages/version-list/components/VersionListTable/VersionListTable.module.scss'

export interface IGenericVersionListTableProps {
  data: ListArtifactVersion
  sortBy: SortByType

  setSortBy: (sortBy: SortByType) => void
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: () => PaginationProps['onPageSizeChange']
}

function GenericVersionListTable({
  data,
  sortBy,
  setSortBy,
  gotoPage,
  onPageSizeChange
}: IGenericVersionListTableProps) {
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const { parent } = useAppStore()
  const [currentSort, currentOrder] = sortBy

  const { artifactVersions = [], itemCount = 0, pageCount = 0, pageIndex = 0, pageSize = 0 } = data || {}
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })

  const columns: Column<ArtifactVersionMetadata>[] = useMemo(() => {
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
        Header: getString('versionList.table.columns.version'),
        accessor: 'name',
        Cell: VersionNameCell,
        serverSortProps: getServerSortProps('name')
      },
      {
        Header: getString('versionList.table.columns.deployments'),
        accessor: 'deployments',
        Cell: VersionDeploymentsCell,
        serverSortProps: getServerSortProps('deployments'),
        hidden: parent === Parent.OSS
      },
      {
        Header: getString('versionList.table.columns.digests'),
        accessor: 'digestCount',
        Cell: VersionDigestsCell,
        serverSortProps: getServerSortProps('digestCount')
      },
      {
        Header: getString('versionList.table.columns.publishedByAt'),
        accessor: 'lastModified', // key from BE
        Cell: VersionPublishedAtCell,
        serverSortProps: getServerSortProps('lastModified')
      }
    ].filter(Boolean) as Column<ArtifactVersionMetadata>[]
  }, [currentSort, currentOrder, getString, setSortBy])

  return (
    <TableV2<ArtifactVersionMetadata>
      className={classNames(css.table, {
        [css.ossTable]: parent === Parent.OSS,
        [css.enterpriseTable]: parent === Parent.Enterprise
      })}
      columns={columns}
      data={artifactVersions}
      pagination={paginationProps}
      sortable
    />
  )
}

export default GenericVersionListTable
