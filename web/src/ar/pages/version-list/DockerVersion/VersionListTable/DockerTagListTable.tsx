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
import { Container, TableV2 } from '@harnessio/uicore'
import type { ArtifactVersionMetadata } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useDecodedParams, useParentHooks } from '@ar/hooks'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import DigestListPage from '@ar/pages/digest-list/DigestListPage'
import { handleToggleExpandableRow } from '@ar/components/TableCells/utils'

import {
  PullCommandCell,
  ToggleAccordionCell,
  VersionActionsCell,
  VersionDeploymentsCell,
  VersionDigestsCell,
  VersionPublishedAtCell
} from '@ar/pages/version-list/components/VersionListTable/VersionListCell'

import type { DockerVersionListTableProps } from './types'
import { DockerVersionNameCell } from './DockerVersionListCell'

import css from './DockerVersionListTable.module.scss'

function DockerTagListTable(props: DockerVersionListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy } = props
  const pathParams = useDecodedParams<ArtifactDetailsPathParams>()
  const { parent } = useAppStore()
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const [expandedRows, setExpandedRows] = React.useState<Set<string>>(new Set())

  const { artifactVersions = [], itemCount = 0, pageCount = 0, pageIndex = 0, pageSize = 0 } = data || {}
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })
  const [currentSort, currentOrder] = sortBy

  const onToggleRow = useCallback((rowData: ArtifactVersionMetadata): void => {
    const value = rowData.name
    setExpandedRows(handleToggleExpandableRow(value))
  }, [])

  const columns: Column<ArtifactVersionMetadata>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: any) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: '',
        width: '30%',
        accessor: 'select',
        id: 'rowSelectOrExpander',
        Cell: ToggleAccordionCell,
        disableSortBy: true,
        expandedRows,
        setExpandedRows
      },
      {
        Header: getString('versionList.table.columns.version'),
        width: '100%',
        accessor: 'name',
        Cell: DockerVersionNameCell,
        serverSortProps: getServerSortProps('name')
      },
      {
        Header: getString('versionList.table.columns.deployments'),
        width: '100%',
        accessor: 'deployments',
        Cell: VersionDeploymentsCell,
        serverSortProps: getServerSortProps('deployments'),
        hidden: parent === Parent.OSS
      },
      {
        Header: getString('versionList.table.columns.digests'),
        width: '100%',
        accessor: 'digestCount',
        Cell: VersionDigestsCell,
        serverSortProps: getServerSortProps('digestCount')
      },
      {
        Header: getString('versionList.table.columns.publishedByAt'),
        width: '100%',
        accessor: 'lastModified',
        Cell: VersionPublishedAtCell,
        serverSortProps: getServerSortProps('lastModified')
      },
      {
        Header: getString('versionList.table.columns.pullCommand'),
        width: '100%',
        accessor: 'pullCommand',
        Cell: PullCommandCell,
        serverSortProps: getServerSortProps('pullCommand')
      },
      {
        Header: '',
        width: '30%',
        accessor: 'actions',
        Cell: VersionActionsCell,
        disableSortBy: true
      }
    ]
      .filter(Boolean)
      .filter(each => !each.hidden) as unknown as Column<ArtifactVersionMetadata>[]
  }, [currentOrder, currentSort, getString, expandedRows])

  const renderRowSubComponent = React.useCallback(
    ({ row }: { row: Row<ArtifactVersionMetadata> }) => (
      <Container className={css.rowSubComponent} onClick={killEvent}>
        <DigestListPage
          repoKey={pathParams.repositoryIdentifier}
          artifact={pathParams.artifactIdentifier}
          version={row.original.name}
        />
      </Container>
    ),
    []
  )

  return (
    <TableV2<ArtifactVersionMetadata>
      className={classNames(css.table)}
      columns={columns}
      data={artifactVersions}
      pagination={paginationProps}
      sortable
      renderRowSubComponent={renderRowSubComponent}
      getRowClassName={row => (expandedRows.has(row.original.name) ? css.activeRow : '')}
      onRowClick={onToggleRow}
      autoResetExpanded={false}
    />
  )
}

export default DockerTagListTable
