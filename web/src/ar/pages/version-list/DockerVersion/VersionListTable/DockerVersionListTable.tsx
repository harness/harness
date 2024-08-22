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
import type { ArtifactVersionMetadata, ListArtifactVersion } from '@harnessio/react-har-service-client'

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
  VersionDeploymentsCell,
  VersionDigestsCell,
  VersionPublishedAtCell
} from '@ar/pages/version-list/components/VersionListTable/VersionListCell'
import { DockerVersionNameCell } from './DockerVersionListCell'
import css from './DockerVersionListTable.module.scss'

export interface DockerVersionListTableProps {
  data: ListArtifactVersion
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

function DockerVersionListTable(props: DockerVersionListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy } = props
  const pathParams = useDecodedParams<ArtifactDetailsPathParams>()
  const { parent } = useAppStore()
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const [expandedRows, setExpandedRows] = React.useState<Set<string>>(new Set())

  const { artifactVersions = [], itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data || {}
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
        accessor: 'select',
        id: 'rowSelectOrExpander',
        Cell: ToggleAccordionCell,
        disableSortBy: true,
        expandedRows,
        setExpandedRows
      },
      {
        Header: getString('versionList.table.columns.version'),
        accessor: 'name',
        Cell: DockerVersionNameCell,
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
        accessor: 'lastModified',
        Cell: VersionPublishedAtCell,
        serverSortProps: getServerSortProps('lastModified')
      },
      {
        Header: getString('versionList.table.columns.pullCommand'),
        accessor: 'pullCommand',
        Cell: PullCommandCell,
        serverSortProps: getServerSortProps('pullCommand')
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
      className={classNames(css.table, {
        [css.ossTable]: parent === Parent.OSS,
        [css.enterpriseTable]: parent === Parent.Enterprise
      })}
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

export default DockerVersionListTable
