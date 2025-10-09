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
import { FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Text } from '@harnessio/uicore'
import type { WebhookExecution } from '@harnessio/react-har-service-client'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'

import { useStrings } from '@ar/frameworks/strings'
import TableCells from '@ar/components/TableCells/TableCells'
import { WebhookTriggerLabelMap } from '@ar/pages/webhook-list/constants'

import ExecutionStatus from '../ExecutionStatus/ExecutionStatus'
import { WebhookExecutionDetailsTab } from '../WebhookExecutionDetailsDrawer/constants'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

export interface WebhookExecutionListColumnActions {
  onSelect?: (val: WebhookExecution, tab: WebhookExecutionDetailsTab) => void
}

type CellType = Renderer<CellTypeWithActions<WebhookExecution>>

export const ExecutionIdCell: CellType = ({ value }) => {
  return (
    <Text icon="execution" iconProps={{ size: 24 }} font={{ variation: FontVariation.BODY }} lineClamp={1}>
      {value}
    </Text>
  )
}

export const ExecutionTriggeredAtCell: CellType = ({ value }) => {
  return <TableCells.LastModifiedCell value={value} />
}

export const ExecutionTriggerTypeCell: CellType = ({ row }) => {
  const { original } = row
  const { triggerType } = original
  const { getString } = useStrings()
  const triggerTypeValue =
    triggerType && WebhookTriggerLabelMap[triggerType] ? getString(WebhookTriggerLabelMap[triggerType]) : triggerType
  return <TableCells.TextCell value={triggerTypeValue} />
}

export const ExecutionPayloadCell: Renderer<{
  row: Row<WebhookExecution>
  column: ColumnInstance<WebhookExecution> & WebhookExecutionListColumnActions
}> = ({ column, row }) => {
  const { onSelect } = column
  const { getString } = useStrings()
  return (
    <Button
      icon="file"
      variation={ButtonVariation.LINK}
      onClick={() => onSelect?.(row.original, WebhookExecutionDetailsTab.Payload)}>
      {getString('view')}
    </Button>
  )
}

export const ExecutionResponseCell: Renderer<{
  row: Row<WebhookExecution>
  column: ColumnInstance<WebhookExecution> & WebhookExecutionListColumnActions
}> = ({ column, row }) => {
  const { onSelect } = column
  const { getString } = useStrings()
  return (
    <Button
      icon="sto-dast"
      variation={ButtonVariation.LINK}
      onClick={() => onSelect?.(row.original, WebhookExecutionDetailsTab.ServerResponse)}>
      {getString('view')}
    </Button>
  )
}

export const ExecutionStatusCell: CellType = ({ value, row }) => {
  return <ExecutionStatus status={value} message={row.original.error} />
}
