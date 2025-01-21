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

import React, { useState } from 'react'
import classNames from 'classnames'
import type { Column } from 'react-table'
import { type PaginationProps, TableV2, useToggleOpen } from '@harnessio/uicore'
import type { ListWebhooksExecutions, WebhookExecution } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useParentHooks } from '@ar/hooks'

import {
  ExecutionIdCell,
  ExecutionPayloadCell,
  ExecutionResponseCell,
  ExecutionStatusCell,
  ExecutionTriggeredAtCell,
  ExecutionTriggerTypeCell
} from './WebhookExecitionListTableCells'

import { WebhookExecutionDetailsTab } from '../WebhookExecutionDetailsDrawer/constants'
import WebhookExecutionDetailsDrawer from '../WebhookExecutionDetailsDrawer/WebhookExecutionDetailsDrawer'

import css from './WebhookExecutionListTable.module.scss'

interface WebhookExecutionListSortBy {
  sort: 'name'
}

export interface WebhookExecutionListTableProps {
  data: ListWebhooksExecutions
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export default function WebhookExecutionListTable(props: WebhookExecutionListTableProps) {
  const { data, gotoPage, onPageSizeChange, sortBy, setSortBy } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const [currentSort, currentOrder] = sortBy || []

  const [selectedExecution, setSelectedExecution] = useState<WebhookExecution | null>(null)
  const [activeTab, setActiveTab] = useState<WebhookExecutionDetailsTab>(WebhookExecutionDetailsTab.Payload)
  const { isOpen, open, close } = useToggleOpen()

  const { executions, itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })

  const handleSelectExecution = (val: WebhookExecution, tab: WebhookExecutionDetailsTab) => {
    setSelectedExecution(val)
    setActiveTab(tab)
    open()
  }

  const handleClose = () => {
    setSelectedExecution(null)
    close()
  }

  const columns: Column<WebhookExecution>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: WebhookExecutionListSortBy) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: getString('webhookExecutionList.table.columns.id'),
        accessor: 'id',
        Cell: ExecutionIdCell,
        serverSortProps: getServerSortProps('id'),
        disableSortBy: true
      },
      {
        Header: getString('webhookExecutionList.table.columns.lastTriggered'),
        accessor: 'created',
        Cell: ExecutionTriggeredAtCell,
        disableSortBy: true
      },
      {
        Header: getString('webhookExecutionList.table.columns.event'),
        accessor: 'triggerType',
        Cell: ExecutionTriggerTypeCell,
        disableSortBy: true
      },
      {
        Header: getString('webhookExecutionList.table.columns.payload'),
        accessor: 'request',
        Cell: ExecutionPayloadCell,
        disableSortBy: true,
        onSelect: handleSelectExecution
      },
      {
        Header: getString('webhookExecutionList.table.columns.response'),
        accessor: 'response',
        Cell: ExecutionResponseCell,
        disableSortBy: true,
        onSelect: handleSelectExecution
      },
      {
        Header: getString('webhookExecutionList.table.columns.status'),
        accessor: 'result',
        Cell: ExecutionStatusCell,
        disableSortBy: true
      }
    ].filter(Boolean) as unknown as Column<WebhookExecution>[]
  }, [currentOrder, currentSort, getString])

  return (
    <>
      <TableV2
        className={classNames(css.table)}
        columns={columns}
        data={executions}
        pagination={paginationProps}
        sortable
        getRowClassName={() => css.tableRow}
      />
      {isOpen && (
        <WebhookExecutionDetailsDrawer
          isOpen={isOpen}
          initialTab={activeTab}
          onClose={handleClose}
          data={selectedExecution}
        />
      )}
    </>
  )
}
