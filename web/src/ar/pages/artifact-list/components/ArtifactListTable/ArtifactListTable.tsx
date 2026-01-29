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
import type { ListVersion, VersionMetadata } from '@harnessio/react-har-service-client'

import { useFeatureFlags, useParentHooks } from '@ar/hooks'
import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import versionFactory from '@ar/frameworks/Version/VersionFactory'
import { handleToggleExpandableRow } from '@ar/components/TableCells/utils'
import ArtifactRowSubComponentWidget from '@ar/frameworks/Version/ArtifactRowSubComponentWidget'
import { SoftDeleteFilterEnum } from '@ar/constants'

import {
  ArtifactDeploymentsCell,
  ArtifactDownloadsCell,
  ArtifactNameCell,
  ArtifactPackageTypeCell,
  ArtifactVersionActions,
  ArtifactVersionCell,
  LatestArtifactCell,
  ToggleAccordionCell,
  ViolationScanStatusCell
} from './ArtifactListTableCell'
import { RepositoryNameCell } from '../RegistryArtifactListTable/RegistryArtifactListTableCell'
import css from './ArtifactListTable.module.scss'

export interface ArtifactListColumnActions {
  refetchList?: () => void
}
export interface ArtifactListTableProps extends ArtifactListColumnActions {
  data: ListVersion
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
  softDeleteFilter?: SoftDeleteFilterEnum
}

export default function ArtifactListTable(props: ArtifactListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy, softDeleteFilter } = props
  const [expandedRows, setExpandedRows] = React.useState<Set<string>>(new Set())
  const { HAR_DEPENDENCY_FIREWALL } = useFeatureFlags()

  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()

  const getRowId = (rowData: VersionMetadata) => {
    return `${rowData.registryIdentifier}/${rowData.package}:${rowData.version}`
  }

  const onToggleRow = useCallback((rowData: VersionMetadata): void => {
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

  const columns: Column<VersionMetadata>[] = React.useMemo(() => {
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
        accessor: 'package',
        Cell: ArtifactNameCell,
        serverSortProps: getServerSortProps('package'),
        width: '200%'
      },
      {
        Header: getString('artifactList.table.columns.version'),
        accessor: 'version',
        Cell: ArtifactVersionCell,
        disableSortBy: true,
        width: '100%'
      },
      ...(HAR_DEPENDENCY_FIREWALL
        ? [
            {
              Header: getString('artifactList.table.columns.scanStatus'),
              accessor: 'scanStatus',
              Cell: ViolationScanStatusCell,
              disableSortBy: true,
              width: '100%'
            }
          ]
        : []),
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
        Header: getString(
          softDeleteFilter === SoftDeleteFilterEnum.ONLY
            ? 'artifactList.table.columns.archivedAt'
            : 'artifactList.table.columns.lastUpdated'
        ),
        accessor: 'lastModified',
        Cell: LatestArtifactCell,
        serverSortProps: getServerSortProps(
          softDeleteFilter === SoftDeleteFilterEnum.ONLY ? 'deletedAt' : 'lastModified'
        ),
        width: '100%'
      },
      {
        Header: '',
        accessor: 'action',
        Cell: ArtifactVersionActions,
        disableSortBy: true,
        width: '30%'
      }
    ].filter(Boolean) as unknown as Column<VersionMetadata>[]
  }, [currentOrder, currentSort, getString, expandedRows, setExpandedRows, softDeleteFilter])

  const renderRowSubComponent = useCallback(
    ({ row }: { row: Row<VersionMetadata> }) => (
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
    <TableV2<VersionMetadata>
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
