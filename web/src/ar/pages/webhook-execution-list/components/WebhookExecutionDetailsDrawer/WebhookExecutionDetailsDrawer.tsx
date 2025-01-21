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
import moment from 'moment'
import { Color, FontVariation } from '@harnessio/design-system'
import { reTriggerWebhookExecution, WebhookExecution } from '@harnessio/react-har-service-client'
import { Drawer, Expander, IDrawerProps, Position, Tab } from '@blueprintjs/core'
import {
  Button,
  ButtonVariation,
  Container,
  getErrorInfoFromErrorObject,
  Layout,
  Tabs,
  Text,
  useToaster
} from '@harnessio/uicore'

import { useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import CommandBlock from '@ar/components/CommandBlock/CommandBlock'
import { prettifyManifestJSON } from '@ar/pages/version-details/utils'
import { WebhookTriggerLabelMap } from '@ar/pages/webhook-list/constants'

import { WebhookExecutionDetailsTab } from './constants'
import ExecutionStatus from '../ExecutionStatus/ExecutionStatus'

import css from './WebhookExecutionDetailsDrawer.module.scss'

interface WebhookExecutionDetailsDrawerProps extends IDrawerProps {
  data: WebhookExecution | null
  initialTab?: WebhookExecutionDetailsTab
}

export default function WebhookExecutionDetailsDrawer(props: WebhookExecutionDetailsDrawerProps) {
  const { data, isOpen, onClose, initialTab } = props
  const [loading, setLoading] = React.useState(false)
  const [activeTab, setActiveTab] = React.useState<WebhookExecutionDetailsTab>(
    initialTab ?? WebhookExecutionDetailsTab.Payload
  )
  const { getString } = useStrings()
  const registryRef = useGetSpaceRef()
  const { showSuccess, showError, clear } = useToaster()

  const handleRetrigger = async () => {
    try {
      setLoading(true)
      await reTriggerWebhookExecution({
        registry_ref: registryRef,
        webhook_execution_id: data?.id?.toString() ?? '',
        webhook_identifier: data?.webhookId?.toString() ?? ''
      })
      showSuccess(getString('webhookExecutionList.retriggerExecutionSuccess'))
    } catch (e) {
      clear()
      showError(getErrorInfoFromErrorObject(e as Error))
    } finally {
      setLoading(false)
    }
  }
  return (
    <Drawer
      position={Position.RIGHT}
      isOpen={isOpen}
      onClose={onClose}
      usePortal
      isCloseButtonShown={false}
      title={
        <Layout.Horizontal
          spacing="large"
          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
          padding={{ top: 'medium', bottom: 'medium' }}>
          <Text icon="execution" iconProps={{ size: 28 }} font={{ variation: FontVariation.H4 }}>
            {data?.id}
          </Text>
          {data?.result && <ExecutionStatus status={data?.result} />}
          <Expander />
          <Button
            disabled={!data?.retriggerable}
            loading={loading}
            onClick={handleRetrigger}
            variation={ButtonVariation.SECONDARY}>
            {getString('webhookExecutionList.retriggerExecution')}
          </Button>
        </Layout.Horizontal>
      }>
      <Container>
        <Layout.Vertical padding="large" spacing="medium">
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Text font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('webhookExecutionList.executionDetailsDrawer.triggeredEvent')}:
            </Text>
            <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.GREY_700}>
              {data?.triggerType && WebhookTriggerLabelMap[data?.triggerType]
                ? getString(WebhookTriggerLabelMap[data?.triggerType])
                : data?.triggerType}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Text font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('webhookExecutionList.executionDetailsDrawer.at')}:
            </Text>
            <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.GREY_700}>
              {moment(data?.created).format(DEFAULT_DATE_TIME_FORMAT)}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Text font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('webhookExecutionList.executionDetailsDrawer.duration')}:
            </Text>
            <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.GREY_700}>
              {getString('webhookExecutionList.executionDetailsDrawer.durationValue', {
                duration: Math.ceil(
                  moment.duration((data?.duration ? data?.duration / 1_000_000 : 0) as number).asSeconds()
                )
              })}
            </Text>
          </Layout.Horizontal>

          {data?.error && (
            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <Text font={{ variation: FontVariation.CARD_TITLE }}>
                {getString('webhookExecutionList.executionDetailsDrawer.error')}:
              </Text>
              <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.ERROR}>
                {data?.error}
              </Text>
            </Layout.Horizontal>
          )}
        </Layout.Vertical>
        <Container className={css.tabsContainer}>
          <Tabs
            id="webhookExecutionDetailsTabs"
            selectedTabId={activeTab}
            onChange={(tabId: WebhookExecutionDetailsTab) => setActiveTab(tabId)}>
            <Tab
              id={WebhookExecutionDetailsTab.Payload}
              title={getString('webhookExecutionList.executionDetailsDrawer.payload')}
              panel={
                <CommandBlock
                  darkmode
                  ignoreWhiteSpaces={false}
                  commandSnippet={prettifyManifestJSON(data?.request ?? {})}
                />
              }
            />
            <Tab
              id={WebhookExecutionDetailsTab.ServerResponse}
              title={getString('webhookExecutionList.executionDetailsDrawer.response')}
              panel={
                <CommandBlock
                  darkmode
                  ignoreWhiteSpaces={false}
                  commandSnippet={prettifyManifestJSON(data?.response ?? {})}
                />
              }
            />
          </Tabs>
        </Container>
      </Container>
    </Drawer>
  )
}
