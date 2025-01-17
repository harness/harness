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
import { Icon, IconProps } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Container, getErrorInfoFromErrorObject, Layout, Text, Toggle, useToaster } from '@harnessio/uicore'
import { updateWebhook, type Trigger, type Webhook } from '@harnessio/react-har-service-client'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'

import { useGetSpaceRef } from '@ar/hooks'
import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import ActionButton from '@ar/components/ActionButton/ActionButton'

import { DefaultStatusIconMap, WebhookStatusIconMap, WebhookTriggerLabelMap } from '../../constants'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<Webhook>>

export interface WebhookListColumnActions {
  refetchList?: () => void
  readonly?: boolean
}

export const WebhookNameCell: Renderer<{
  row: Row<Webhook>
  column: ColumnInstance<Webhook> & WebhookListColumnActions
}> = ({ row, column }) => {
  const { name, enabled, identifier } = row.original
  const { readonly } = column
  const [isEnabled, setIsEnabled] = useState(enabled)
  const registryRef = useGetSpaceRef()

  const { showError, clear } = useToaster()

  const handleUpdateToggle = async (checked: boolean) => {
    setIsEnabled(checked)
    try {
      await updateWebhook({
        registry_ref: registryRef,
        webhook_identifier: identifier,
        body: {
          ...row.original,
          enabled: checked
        }
      })
    } catch (e) {
      clear()
      showError(getErrorInfoFromErrorObject(e as Error))
      setIsEnabled(enabled)
    }
  }

  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      <Icon name="code-webhook" size={24} />
      <Container onClick={killEvent}>
        <Toggle disabled={readonly} checked={isEnabled} onToggle={handleUpdateToggle} />
      </Container>
      <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.PRIMARY_7}>
        {name}
      </Text>
    </Layout.Horizontal>
  )
}

export const WebhookTriggerCell: CellType = ({ value }) => {
  const { getString } = useStrings()
  const getDisplayText = () => {
    if (!value || !value.length) return getString('all')
    return value
      .map((each: Trigger) => {
        if (WebhookTriggerLabelMap[each]) return getString(WebhookTriggerLabelMap[each])
        return each
      })
      .join(', ')
  }
  return (
    <Text lineClamp={2} font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
      {getDisplayText()}
    </Text>
  )
}

export const WebhookStatusCell: CellType = ({ row }) => {
  const { original } = row
  const { latestExecutionResult } = original
  const iconProps: IconProps =
    latestExecutionResult && WebhookStatusIconMap[latestExecutionResult]
      ? WebhookStatusIconMap[latestExecutionResult]
      : DefaultStatusIconMap
  return <Icon {...iconProps} size={18} />
}

export const WebhookActionsCell: CellType = () => {
  const [open, setOpen] = useState(false)
  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {/* TODO: Add webhook actions here */}
      <>Option 1</>
    </ActionButton>
  )
}
