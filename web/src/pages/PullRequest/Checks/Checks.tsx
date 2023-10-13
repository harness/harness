/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useMemo, useState } from 'react'
import { Falsy, Match, Render, Truthy } from 'react-jsx-match'
import { get } from 'lodash-es'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { Container, Layout, Text, FlexExpander, Button, ButtonVariation, ButtonSize } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { LogViewer } from 'components/LogViewer/LogViewer'
import { PullRequestCheckType } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { Split } from 'components/Split/Split'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import type { TypesCheck, TypesStage } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import type { ChecksProps } from './ChecksUtils'
import { CheckPipelineSteps } from './CheckPipelineSteps'
import { ChecksMenu } from './ChecksMenu'
import css from './Checks.module.scss'

export const Checks: React.FC<ChecksProps> = ({ repoMetadata, pullRequestMetadata, prChecksDecisionResult }) => {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const [selectedItemData, setSelectedItemData] = useState<TypesCheck>()
  const [selectedStage, setSelectedStage] = useState<TypesStage | null>(null)
  const isCheckDataMarkdown = useMemo(
    () => selectedItemData?.payload?.kind === PullRequestCheckType.MARKDOWN,
    [selectedItemData?.payload?.kind]
  )
  const logContent = useMemo(
    () => get(selectedItemData, 'payload.data.details', selectedItemData?.summary || ''),
    [selectedItemData]
  )
  const executionLink = useMemo(() => {
    if (selectedStage) {
      return routes.toCODEExecution({
        repoPath: repoMetadata?.path as string,
        pipeline: selectedItemData?.uid as string,
        execution: get(selectedItemData, 'payload.data.execution_number', '')
      })
    } else {
      return selectedItemData?.link
    }
  }, [repoMetadata?.path, routes, selectedItemData, selectedStage])

  if (!prChecksDecisionResult) {
    return null
  }

  return (
    <Container className={css.main}>
      <Match expr={prChecksDecisionResult?.overallStatus}>
        <Truthy>
          <Split split="vertical" size={400} minSize={300} maxSize={700} primary="first">
            <ChecksMenu
              repoMetadata={repoMetadata}
              pullRequestMetadata={pullRequestMetadata}
              prChecksDecisionResult={prChecksDecisionResult}
              onDataItemChanged={data => {
                setTimeout(() => setSelectedItemData(data), 0)
              }}
              setSelectedStage={setSelectedStage}
            />
            <Container
              className={cx(css.content, {
                [css.markdown]: isCheckDataMarkdown,
                [css.terminal]: !isCheckDataMarkdown
              })}>
              <Render when={selectedItemData}>
                <Container className={css.header}>
                  <Layout.Horizontal className={css.headerLayout} spacing="small">
                    <ExecutionStatus
                      className={cx(css.status, {
                        [css.invert]: selectedItemData?.status === ExecutionState.PENDING
                      })}
                      status={selectedItemData?.status as ExecutionState}
                      iconSize={20}
                      noBackground
                      iconOnly
                    />
                    <Text
                      font={{ variation: FontVariation.BODY1 }}
                      color={Color.WHITE}
                      lineClamp={1}
                      tooltipProps={{ portalClassName: css.popover }}>
                      {selectedItemData?.uid}
                      {selectedStage ? ` / ${selectedStage.name}` : ''}
                    </Text>
                    <FlexExpander />
                    <Render when={executionLink}>
                      <Button
                        className={css.noShrink}
                        text={getString('prChecks.viewExternal')}
                        rightIcon="chevron-right"
                        variation={ButtonVariation.SECONDARY}
                        size={ButtonSize.SMALL}
                        onClick={() => {
                          if (selectedStage) {
                            history.push(executionLink as string)
                          } else {
                            window.open(executionLink, '_blank')
                          }
                        }}
                      />
                    </Render>
                  </Layout.Horizontal>
                </Container>
              </Render>
              <Match expr={isCheckDataMarkdown}>
                <Truthy>
                  <Container className={css.markdownContainer}>
                    <MarkdownViewer darkMode source={logContent} />
                  </Container>
                </Truthy>
                <Falsy>
                  <Match expr={selectedStage}>
                    <Truthy>
                      <CheckPipelineSteps
                        repoMetadata={repoMetadata}
                        pullRequestMetadata={pullRequestMetadata}
                        pipelineName={selectedItemData?.uid as string}
                        stage={selectedStage as TypesStage}
                        executionNumber={get(selectedItemData, 'payload.data.execution_number', '')}
                      />
                    </Truthy>
                    <Falsy>
                      <LogViewer content={logContent} className={css.logViewer} />
                    </Falsy>
                  </Match>
                </Falsy>
              </Match>
            </Container>
          </Split>
        </Truthy>
        <Falsy>
          <Container flex={{ align: 'center-center' }} height="90%">
            <Text font={{ variation: FontVariation.BODY1 }}>{getString('prChecks.notFound')}</Text>
          </Container>
        </Falsy>
      </Match>
    </Container>
  )
}
