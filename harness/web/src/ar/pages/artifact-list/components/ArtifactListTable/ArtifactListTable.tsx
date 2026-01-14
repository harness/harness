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
import type { ArtifactMetadata, ListArtifact } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import versionFactory from '@ar/frameworks/Version/VersionFactory'
import { handleToggleExpandableRow } from '@ar/components/TableCells/utils'
import ArtifactRowSubComponentWidget from '@ar/frameworks/Version/ArtifactRowSubComponentWidget'

import {
  ArtifactDeploymentsCell,
  ArtifactDownloadsCell,
  ArtifactNameCell,
  ArtifactPackageTypeCell,
  ArtifactVersionActions,
  ArtifactVersionCell,
  LatestArtifactCell,
  ToggleAccordionCell
} from './ArtifactListTableCell'
import { RepositoryNameCell } from '../RegistryArtifactListTable/RegistryArtifactListTableCell'
import css from './ArtifactListTable.module.scss'

export interface ArtifactListColumnActions {
  refetchList?: () => void
}
export interface ArtifactListTableProps extends ArtifactListColumnActions {
  data: ListArtifact
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export default function ArtifactListTable(props: ArtifactListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy } = props
  const [expandedRows, setExpandedRows] = React.useState<Set<string>>(new Set())

  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()

  const getRowId = (rowData: ArtifactMetadata) => {
    return `${rowData.registryIdentifier}/${rowData.name}:${rowData.version}`
  }

  const onToggleRow = useCallback((rowData: ArtifactMetadata): void => {
    const value = getRowId(rowData)
    const repositoryType = versionFactory?.getVersionType(rowData.packageType)
    if (!repositoryType?.getHasArtifactRowSubComponent()) return
    setExpandedRows(handleToggleExpandableRow(value))
  }, [])

  const { artifacts = [], itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data || {}
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })
  const [currentSort, currentOrder] = sortBy

  const columns: Column<ArtifactMetadata>[] = React.useMemo(() => {
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
        setExpandedRows,
        getRowId,
        width: '30%'
      },
      {
        Header: getString('artifactList.table.columns.artifactName'),
        accessor: 'name',
        Cell: ArtifactNameCell,
        serverSortProps: getServerSortProps('name'),
        width: '200%'
      },
      {
        Header: getString('artifactList.table.columns.version'),
        accessor: 'version',
        Cell: ArtifactVersionCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('artifactList.table.columns.repository'),
        accessor: 'registryIdentifier',
        Cell: RepositoryNameCell,
        serverSortProps: getServerSortProps('registryIdentifier'),
        width: '100%'
      },
      {
        Header: getString('artifactList.table.columns.type'),
        accessor: 'packageType',
        Cell: ArtifactPackageTypeCell,
        serverSortProps: getServerSortProps('packageType'),
        width: '100%'
      },
      {
        Header: getString('artifactList.table.columns.downloads'),
        accessor: 'downloadsCount',
        Cell: ArtifactDownloadsCell,
        serverSortProps: getServerSortProps('downloadsCount'),
        width: '100%'
      },
      {
        Header: getString('artifactList.table.columns.environments'),
        accessor: 'environments',
        Cell: ArtifactDeploymentsCell,
        disableSortBy: true,
        width: '100%'
      },
      {
        Header: getString('artifactList.table.columns.lastUpdated'),
        accessor: 'lastUpdated',
        Cell: LatestArtifactCell,
        serverSortProps: getServerSortProps('lastUpdated'),
        width: '100%'
      },
      {
        Header: '',
        accessor: 'action',
        Cell: ArtifactVersionActions,
        disableSortBy: true,
        width: '30%'
      }
    ].filter(Boolean) as unknown as Column<ArtifactMetadata>[]
  }, [currentOrder, currentSort, getString, expandedRows, setExpandedRows])

  const renderRowSubComponent = useCallback(
    ({ row }: { row: Row<ArtifactMetadata> }) => (
      <Container className={css.tableRowSubComponent} onClick={killEvent}>
        <ArtifactRowSubComponentWidget
          packageType={row.original.packageType as RepositoryPackageType}
          data={row.original}
        />
      </Container>
    ),
    []
  )

  return (
    <TableV2<ArtifactMetadata>
      className={classNames(css.table)}
      columns={columns}
      data={artifacts}
      pagination={paginationProps}
      sortable
      renderRowSubComponent={renderRowSubComponent}
      getRowClassName={row => classNames(css.tableRow, { [css.activeRow]: expandedRows.has(getRowId(row.original)) })}
      onRowClick={onToggleRow}
      autoResetExpanded={false}
    />
  )
}
