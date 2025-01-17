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
import type { Column } from 'react-table'
import { useHistory } from 'react-router-dom'
import { type PaginationProps, TableV2 } from '@harnessio/uicore'
import type { ListWebhooks, Webhook } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useParentHooks, useRoutes } from '@ar/hooks'

import {
  WebhookActionsCell,
  WebhookListColumnActions,
  WebhookNameCell,
  WebhookStatusCell,
  WebhookTriggerCell
} from './WebhookListTableCells'

import css from './WebhookListTable.module.scss'

interface WebhookListSortBy {
  sort: 'name'
}

export interface WebhookListTableProps extends WebhookListColumnActions {
  data: ListWebhooks
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: PaginationProps['onPageSizeChange']
  setSortBy: (sortBy: string[]) => void
  sortBy: string[]
  minimal?: boolean
}

export default function WebhookListTable(props: WebhookListTableProps): JSX.Element {
  const { data, gotoPage, onPageSizeChange, readonly, sortBy, setSortBy } = props
  const { useDefaultPaginationProps } = useParentHooks()
  const { getString } = useStrings()
  const history = useHistory()
  const routes = useRoutes()
  const [currentSort, currentOrder] = sortBy || []

  const { webhooks, itemCount = 0, pageCount = 0, pageIndex, pageSize = 0 } = data
  const paginationProps = useDefaultPaginationProps({
    itemCount,
    pageSize,
    pageCount,
    pageIndex,
    gotoPage,
    onPageSizeChange
  })

  const columns: Column<Webhook>[] = React.useMemo(() => {
    const getServerSortProps = (id: string) => {
      return {
        enableServerSort: true,
        isServerSorted: currentSort === id,
        isServerSortedDesc: currentOrder === 'DESC',
        getSortedColumn: ({ sort }: WebhookListSortBy) => {
          setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
        }
      }
    }
    return [
      {
        Header: getString('webhookList.table.columns.name'),
        accessor: 'name',
        Cell: WebhookNameCell,
        serverSortProps: getServerSortProps('name'),
        readonly: readonly
      },
      {
        Header: getString('webhookList.table.columns.trigger'),
        accessor: 'triggers',
        Cell: WebhookTriggerCell,
        disableSortBy: true
      },
      {
        Header: '',
        accessor: 'latestExecutionResult',
        Cell: WebhookStatusCell,
        disableSortBy: true
      },
      {
        Header: '',
        accessor: 'menu',
        Cell: WebhookActionsCell,
        disableSortBy: true
      }
    ].filter(Boolean) as unknown as Column<Webhook>[]
  }, [currentOrder, currentSort, getString, readonly])

  return (
    <TableV2
      className={classNames(css.table)}
      columns={columns}
      data={webhooks}
      pagination={paginationProps}
      sortable
      getRowClassName={() => css.tableRow}
      onRowClick={rowDetails => {
        history.push(
          routes.toARRepositoryWebhookDetails({
            repositoryIdentifier: rowDetails.identifier,
            webhookIdentifier: rowDetails.identifier
          })
        )
      }}
    />
  )
}
