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

import React, { useEffect, useMemo, useReducer, useState } from 'react'
import { Falsy, Match, Render, Truthy } from 'react-jsx-match'
import { get } from 'lodash-es'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { Container, Layout, Text, FlexExpander, Button, ButtonVariation, ButtonSize } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { LogViewer } from 'components/LogViewer/LogViewer'
import { PullRequestCheckType, PullRequestSection } from 'utils/Utils'
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

// Define the reducer outside your component
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mapReducer = (state: any, action: { type: any; newEntries: any }) => {
  switch (action.type) {
    case 'UPDATE_MAP':
      // Return a new updated map
      return new Map([...state, ...action.newEntries])
    case 'RESET_MAP':
      return new Map()
    default:
      return state
  }
}

interface SelectedItemDataInterface {
  execution_id: string
  stage_id: string
}

export const Checks: React.FC<ChecksProps> = ({ repoMetadata, pullReqMetadata, prChecksDecisionResult }) => {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes, standalone } = useAppContext()
  const [selectedItemData, setSelectedItemData] = useState<TypesCheck>()
  const [selectedStage, setSelectedStage] = useState<TypesStage | null>(null)
  const [queue, setQueue] = useState<string[]>([])
  const [stepNameLogKeyMap, dispatch] = useReducer(mapReducer, new Map())
  const { hooks } = useAppContext()
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
        pipeline: selectedItemData?.identifier as string,
        execution: get(selectedItemData, 'payload.data.execution_number', '')
      })
    } else {
      return selectedItemData?.link
    }
  }, [repoMetadata?.path, routes, selectedItemData, selectedStage])
  const executionId =
    standalone && selectedItemData ? null : (selectedItemData?.payload?.data as SelectedItemDataInterface)?.execution_id
  const selectedStageId =
    standalone && selectedItemData ? null : (selectedItemData?.payload?.data as SelectedItemDataInterface)?.stage_id

  useEffect(() => {
    // If there is a new selectedStageId, reset the map
    if (selectedStageId) {
      dispatch({ type: 'RESET_MAP', newEntries: {} })
    }
  }, [selectedStageId])
  const hookData = hooks?.useExecutionDataHook?.(executionId, selectedStageId)
  const executionApiCallData = hookData?.data
  const rootNodeId = executionApiCallData?.data?.executionGraph?.rootNodeId

  useEffect(() => {
    if (rootNodeId) {
      enqueue(rootNodeId)
    }
  }, [rootNodeId])

  useEffect(() => {
    if (queue.length !== 0) {
      processExecutionData(queue)
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queue, selectedStageId, hookData])
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const enqueue = (item: any) => {
    setQueue(prevQueue => [...prevQueue, item])
  }
  useEffect(() => {
    const pollingInterval = 4000

    const fetchAndProcessData = () => {
      hookData?.refetch()
      if (hookData && rootNodeId) {
        enqueue(rootNodeId) // Use your existing enqueue logic here
      }
    }
    // Set up the polling with setInterval
    const intervalId = setInterval(fetchAndProcessData, pollingInterval)
    // Clean up the interval on component unmount
    return () => clearInterval(intervalId) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hookData, enqueue, dispatch])

  if (!prChecksDecisionResult) {
    return null
  }

  const processExecutionData = (curQueue: string[]) => {
    const newQueue = [...curQueue]
    const newEntries = []
    while (newQueue.length !== 0) {
      const item = newQueue.shift()
      if (item) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const nodeArray = (executionApiCallData.data.executionGraph.nodeAdjacencyListMap as any)[item]
        //add node to queue
        if (nodeArray) {
          nodeArray.children.map((node: string) => {
            newQueue.push(node)
          })
        }
        if (nodeArray) {
          nodeArray.nextIds.map((node: string) => {
            newQueue.push(node)
          })
        }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const nodeMapItem = (executionApiCallData.data.executionGraph.nodeMap as any)[item]
        if (nodeMapItem && nodeMapItem?.stepParameters) {
          // Assume that you generate a key-yarn value pair for the map here
          const key = nodeMapItem.stepParameters.name ? nodeMapItem.stepParameters.name : ''
          const logBaseKey = nodeMapItem.logBaseKey
          const status = nodeMapItem.status
          const timeStart = nodeMapItem.startTs
          const timeEnd = nodeMapItem.endTs

          if (item !== rootNodeId) {
            newEntries.push([key, { status, logBaseKey, timeStart, timeEnd }])
          }
        } else {
          continue
        }
      }
    }
    // Update the map using the reducer
    if (newEntries.length > 0) {
      dispatch({ type: 'UPDATE_MAP', newEntries })
    }

    setQueue(newQueue) // Update the queue state
  }

  return (
    <Container className={css.main} data-page-section={PullRequestSection.CHECKS}>
      <Match expr={prChecksDecisionResult?.overallStatus}>
        <Truthy>
          <Split split="vertical" size={400} minSize={300} maxSize={700} primary="first">
            <ChecksMenu
              repoMetadata={repoMetadata}
              pullReqMetadata={pullReqMetadata}
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
                      {selectedItemData?.identifier}
                      {selectedStage ? ` / ${selectedStage.name}` : ''}
                    </Text>
                    <FlexExpander />
                    <Render when={executionLink}>
                      <Button
                        className={css.noShrink}
                        text={getString('prChecks.viewExternal')}
                        rightIcon="chevron-right"
                        variation={ButtonVariation.TERTIARY}
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
                        pullReqMetadata={pullReqMetadata}
                        pipelineName={selectedItemData?.identifier as string}
                        stage={selectedStage as TypesStage}
                        executionNumber={get(selectedItemData, 'payload.data.execution_number', '')}
                      />
                    </Truthy>
                    <Falsy>
                      <LogViewer
                        content={logContent}
                        stepNameLogKeyMap={stepNameLogKeyMap}
                        className={css.logViewer}
                        setSelectedStage={setSelectedStage}
                        selectedItemData={selectedItemData}
                      />
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
