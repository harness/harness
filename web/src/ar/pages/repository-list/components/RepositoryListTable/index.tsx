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
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import type { Column } from 'react-table'
import { TableV2, type PaginationProps } from '@harnessio/uicore'
import type { ListRegistry, RegistryMetadata } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useFeatureFlags, useParentHooks, useRoutes } from '@ar/hooks'
import { useParentUtils } from '@ar/hooks/useParentUtils'
import useGetScopeFromRegistryPath from '@ar/pages/repository-details/hooks/useGetScopeFromRegistryPath/useGetScopeFromRegistryPath'

import {
  LastModifiedCell,
  RepositoryActionsCell,
  RepositoryArtifactsCell,
  RepositoryDownloadsCell,
  RepositoryLocationBadgeCell,
  RepositoryNameCell,
  RepositoryScopeCell,
  RepositorySizeCell,
  RepositoryUrlCell
} from './RepositoryListCells'
import type { RepositoryListSortBy } from './types'
import css from './RepositoryListTable.module.scss'

export interface RepositoryListColumnActions {
  refetchList?: () => void
}
export interface RepositoryListTableProps extends RepositoryListColumnActions {
  data: ListRegistry
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
  showScope?: boolean
}

export function RepositoryListTable(props: RepositoryListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy, showScope } = props
  const { HAR_REGISTRY_SCOPE_FILTER } = useFeatureFlags()
  const { useDefaultPaginationProps } = useParentHooks()
  const { routeToRegistryDetails } = useParentUtils()
  const routes = useRoutes()
  const { getString } = useStrings()
  const history = useHistory()

  const { registries, itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })
  const [currentSort, currentOrder] = sortBy

  const { getScopeFromRegistryPath } = useGetScopeFromRegistryPath()

  const handleNavigateToRegistryDetails = (rowDetails: RegistryMetadata) => {
    if (HAR_REGISTRY_SCOPE_FILTER) {
      history.push(
        routeToRegistryDetails({
          ...getScopeFromRegistryPath(rowDetails.path),
          module: 'har',
          repositoryIdentifier: rowDetails.identifier
        })
      )
    } else {
      history.push(
        routes.toARRepositoryDetails({
          repositoryIdentifier: rowDetails.identifier
        })
      )
    }
  }

  const columns: Column<RegistryMetadata>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: RepositoryListSortBy) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: getString('repositoryList.table.columns.nameAndEnvironment'),
        accessor: 'identifier',
        Cell: RepositoryNameCell,
        serverSortProps: getServerSortProps('identifier')
      },
      showScope &&
        HAR_REGISTRY_SCOPE_FILTER && {
          Header: '',
          accessor: 'path',
          Cell: RepositoryScopeCell,
          disableSortBy: true
        },
      {
        Header: getString('repositoryList.table.columns.type'),
        accessor: 'type',
        Cell: RepositoryLocationBadgeCell,
        serverSortProps: getServerSortProps('type')
      },
      {
        Header: getString('repositoryList.table.columns.size'),
        accessor: 'registrySize',
        Cell: RepositorySizeCell,
        serverSortProps: getServerSortProps('registrySize')
      },
      {
        Header: getString('repositoryList.table.columns.artifacts'),
        accessor: 'artifactsCount',
        Cell: RepositoryArtifactsCell,
        serverSortProps: getServerSortProps('artifactsCount')
      },
      {
        Header: getString('repositoryList.table.columns.downloads'),
        accessor: 'downloadsCount',
        Cell: RepositoryDownloadsCell,
        serverSortProps: getServerSortProps('downloadsCount')
      },
      {
        Header: getString('repositoryList.table.columns.lastModified'),
        accessor: 'lastModified',
        Cell: LastModifiedCell,
        serverSortProps: getServerSortProps('lastModified')
      },
      {
        Header: '',
        accessor: 'url',
        Cell: RepositoryUrlCell,
        disableSortBy: true
      },
      {
        Header: '',
        accessor: 'menu',
        Cell: RepositoryActionsCell,
        disableSortBy: true
      }
    ].filter(Boolean) as unknown as Column<RegistryMetadata>[]
  }, [currentOrder, currentSort, getString, showScope, HAR_REGISTRY_SCOPE_FILTER])

  return (
    <TableV2
      className={cx(css.table, css.alignColumns, {
        [css.scopeColumn]: showScope
      })}
      columns={columns}
      data={registries}
      pagination={paginationProps}
      sortable
      getRowClassName={() => css.tableRow}
      onRowClick={handleNavigateToRegistryDetails}
    />
  )
}
