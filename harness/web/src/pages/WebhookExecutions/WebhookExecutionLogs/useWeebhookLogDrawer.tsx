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

import React, { useMemo, useState } from 'react'
import { Drawer, Position } from '@blueprintjs/core'
import cx from 'classnames'
import {
  Button,
  ButtonVariation,
  Container,
  Layout,
  PageBody,
  PageHeader,
  Tabs,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import MonacoEditor from 'react-monaco-editor'
import { defaultTo, isEmpty } from 'lodash-es'
import { useMutate } from 'restful-react'
import moment from 'moment'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { getErrorMessage } from 'utils/Utils'
import { getConfig } from 'services/config'
import { DateTimeWithLocalContentInline } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { CodeIcon, ExecutionTabs, WebhookIndividualEvent, getEventDescription } from 'utils/GitUtils'
import type { RepoRepositoryOutput, TypesWebhookExecution, TypesWebhookExecutionResponse } from 'services/code'
import { useModalHook } from 'hooks/useModalHook'
import css from './useWebhookLogDrawer.module.scss'

interface LogViewerProps {
  data?: string | TypesWebhookExecutionResponse
}

export function useWeebhookLogDrawer(refetchExecutionList: () => Promise<void>) {
  const [executionData, setExecutionData] = useState<TypesWebhookExecution>()
  const [activeTab, setActiveTab] = useState<string>(ExecutionTabs.PAYLOAD)
  const [path, setPath] = useState('')
  const { mutate: retriggerExection } = useMutate({
    verb: 'POST',
    base: getConfig('code/api/v1'),
    path
  })
  const { getString } = useStrings()

  const LogViewer = (props: LogViewerProps) => {
    const { data } = props
    return (
      <Container padding={'medium'} className={css.logsContainer}>
        <CopyButton
          content={JSON.stringify(data, null, 2)}
          className={css.copyButton}
          icon={CodeIcon.Copy}
          color={Color.PRIMARY_7}
          iconProps={{ size: 20 }}
        />
        <MonacoEditor
          className={css.editor}
          height={'100vh'}
          language="json"
          value={JSON.stringify(data, null, 2)}
          data-testid="monaco-editor"
          theme="vs-dark"
          options={{
            fontFamily: "'Roboto Mono', monospace",
            fontSize: 13,
            scrollBeyondLastLine: true,
            minimap: {
              enabled: false
            },
            unicodeHighlight: {
              ambiguousCharacters: false
            },
            lineNumbers: 'on',
            glyphMargin: true,
            folding: false,
            lineDecorationsWidth: 60,
            wordWrap: 'on',
            scrollbar: {
              verticalScrollbarSize: 0
            },
            renderLineHighlight: 'none',
            wordWrapBreakBeforeCharacters: '',
            lineNumbersMinChars: 0,
            wordBasedSuggestions: 'off',
            readOnly: true
          }}
        />
      </Container>
    )
  }

  const tabListArray = useMemo(
    () => [
      {
        id: ExecutionTabs.PAYLOAD,
        title: ExecutionTabs.PAYLOAD,
        panel: <LogViewer data={JSON.parse(executionData?.request?.body ?? '{}')} />
      },
      {
        id: ExecutionTabs.SERVER_RESPONSE,
        title: ExecutionTabs.SERVER_RESPONSE,
        panel: <LogViewer data={executionData?.response} />
      }
    ],
    [activeTab, executionData]
  )
  const { showSuccess, showError } = useToaster()
  const [openModal, hideModal] = useModalHook(
    () => (
      <Drawer position={Position.RIGHT} isOpen={true} isCloseButtonShown={true} size={'50%'} onClose={hideModal}>
        <PageHeader
          title={
            <Text icon={'execution'} iconProps={{ size: 24 }} font={{ variation: FontVariation.H4 }}>
              {executionData?.id}
            </Text>
          }
          content={
            <Button
              disabled={!executionData?.retriggerable}
              tooltipProps={{
                disabled: executionData?.retriggerable,
                position: Position.TOP,
                interactionKind: 'hover'
              }}
              tooltip={getString('notRetriggerableMessage')}
              variation={ButtonVariation.SECONDARY}
              onClick={() =>
                retriggerExection({})
                  .then(() => {
                    showSuccess(getString('reTriggeredExecution'))
                    refetchExecutionList()
                  })
                  .catch(err => {
                    showError(getErrorMessage(err))
                  })
              }>
              {getString('retriggerExecution')}
            </Button>
          }
        />

        <PageBody className={cx(css.pageBody)}>
          <Layout.Vertical className={css.executionContext}>
            <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <Text font={{ variation: FontVariation.H6 }}>{getString('triggeredEvent')}: </Text>
              <Text color={Color.GREY_600}>
                {getEventDescription(executionData?.trigger_type as WebhookIndividualEvent)}
              </Text>
            </Layout.Horizontal>
            <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <Text font={{ variation: FontVariation.H6 }}>{getString('atSubTitle')}: </Text>
              <Container>
                <DateTimeWithLocalContentInline time={defaultTo(executionData?.created as number, 0)} />
              </Container>
            </Layout.Horizontal>
            <Layout.Horizontal spacing={'small'}>
              <Text font={{ variation: FontVariation.H6 }}>Duration:</Text>
              <Text color={Color.GREY_600}>
                {Math.ceil(
                  moment
                    .duration((executionData?.duration ? executionData?.duration / 1_000_000 : 0) as number)
                    .asSeconds()
                )}
                {'s'}
              </Text>
            </Layout.Horizontal>
          </Layout.Vertical>
          <Render when={!isEmpty(executionData?.error)}>
            <Container
              intent="danger"
              background="red100"
              border={{
                color: 'red500'
              }}
              margin={{ top: 'small', right: 'medium', left: 'medium' }}>
              <Text
                className={css.errorMessage}
                icon="error-outline"
                iconProps={{ size: 16, margin: { right: 'small' } }}
                padding={{ left: 'large', right: 'large', top: 'small', bottom: 'small' }}
                color={Color.ERROR}>
                {executionData?.error}
              </Text>
            </Container>
          </Render>
          <Container className={cx(css.main, css.tabsContainer)}>
            <Tabs
              id="WebhookExecutionLogs"
              large={false}
              defaultSelectedTabId={activeTab}
              animate={false}
              onChange={(id: string) => {
                setActiveTab(id)
              }}
              tabList={tabListArray}></Tabs>
          </Container>
        </PageBody>
      </Drawer>
    ),
    [executionData, activeTab, path]
  )

  const openExecutionLogs = (
    webhookExecution: TypesWebhookExecution,
    logTab: ExecutionTabs,
    repoMetadata?: RepoRepositoryOutput
  ) => {
    setExecutionData(webhookExecution)
    setActiveTab(logTab)
    setPath(
      `/repos/${repoMetadata?.path}/+/webhooks/${webhookExecution.webhook_id}/executions/${webhookExecution.id}/retrigger`
    )
    openModal()
  }

  return { openModal, hideModal, openExecutionLogs }
}
