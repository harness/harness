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
import { defaultTo } from 'lodash-es'
import type { Column } from 'react-table'
import { TableV2, type PaginationProps } from '@harnessio/uicore'
import type { ArtifactDeploymentsDetail, ArtifactDeploymentsDetails } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import type { DockerVersionDeploymentsListSortBy } from './types'
import {
  DeploymentPipelineCell,
  EnvironmentNameCell,
  EnvironmentTypeCell,
  LastModifiedCell,
  ServiceListCell
} from './DeploymentsTableCells'

import css from './DeploymentsTable.module.scss'

interface DockerVersionDeploymentsTableProps {
  data: ArtifactDeploymentsDetails
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
}

export default function DockerVersionDeploymentsTable(props: DockerVersionDeploymentsTableProps) {
  const { data, gotoPage, onPageSizeChange, setSortBy, sortBy } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const { deployments } = data

  const paginationProps = useDefaultPaginationProps({
    itemCount: defaultTo(deployments.itemCount, 0),
    pageSize: defaultTo(deployments.pageSize, 0),
    pageCount: defaultTo(deployments.pageCount, 0),
    pageIndex: defaultTo(deployments.pageIndex, 0),
    gotoPage,
    onPageSizeChange
  })
  const [currentSort, currentOrder] = sortBy

  const columns: Column<ArtifactDeploymentsDetail>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: DockerVersionDeploymentsListSortBy) => {
          if (!sort) return
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: getString('versionDetails.deploymentsTable.columns.environment'),
        accessor: 'envName',
        Cell: EnvironmentNameCell,
        serverSortProps: getServerSortProps('envName')
      },
      {
        Header: getString('versionDetails.deploymentsTable.columns.type'),
        accessor: 'envType',
        Cell: EnvironmentTypeCell,
        serverSortProps: getServerSortProps('envType')
      },
      {
        Header: getString('versionDetails.deploymentsTable.columns.services'),
        accessor: 'serviceName',
        Cell: ServiceListCell,
        serverSortProps: getServerSortProps('serviceName')
      },
      {
        Header: getString('versionDetails.deploymentsTable.columns.deploymentPipeline'),
        accessor: 'lastPipelineExecutionName',
        Cell: DeploymentPipelineCell,
        serverSortProps: getServerSortProps('lastPipelineExecutionName')
      },
      {
        Header: getString('versionDetails.deploymentsTable.columns.triggeredBy'),
        accessor: 'lastDeployedAt',
        Cell: LastModifiedCell,
        serverSortProps: getServerSortProps('lastDeployedAt')
      }
    ].filter(Boolean) as unknown as Column<ArtifactDeploymentsDetail>[]
  }, [currentOrder, currentSort, getString])

  return (
    <TableV2
      className={classNames(css.table)}
      columns={columns}
      data={defaultTo(deployments.deployments, [])}
      pagination={paginationProps}
      sortable
    />
  )
}
