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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import Anser from 'anser'
import cx from 'classnames'
import { Button, ButtonSize, ButtonVariation, Container, FlexExpander, Layout, Text, Utils } from '@harnessio/uicore'
import { NavArrowRight } from 'iconoir-react'
import { isEmpty } from 'lodash-es'
import { Color, FontVariation } from '@harnessio/design-system'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import { capitalizeFirstLetter, parseLogString } from 'pages/PullRequest/Checks/ChecksUtils'
import { useAppContext } from 'AppContext'
import { ButtonRoleProps, timeDistance } from 'utils/Utils'
import { useScheduleJob } from 'hooks/useScheduleJob'
import type { EnumCheckPayloadKind, TypesCheck, TypesStage } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import css from './LogViewer.module.scss'
export interface LogViewerProps {
  search?: string
  content?: string
  className?: string
  stepNameLogKeyMap?: Map<string, { status: string; logBaseKey: string; timeStart: number; timeEnd: number }>
  setSelectedStage: React.Dispatch<React.SetStateAction<TypesStage | null>>
  selectedItemData: TypesCheck | undefined
}

export interface LogLine {
  time: string
  message: string
  out: string
  level: string
  details: {
    [key: string]: string
  }
  pos: number
  logLevel: string
}
enum StepTypes {
  LITEENGINETASK = 'liteEngineTask',
  INITIALIZE = 'Initialize'
}

enum StepStatus {
  RUNNING = 'running',
  SUCCESS = 'success'
}

export type EnumCheckPayloadKindExtended = EnumCheckPayloadKind | 'harness_stage'

const LogTerminal: React.FC<LogViewerProps> = ({
  content,
  className,
  stepNameLogKeyMap,
  setSelectedStage,
  selectedItemData
}) => {
  const { hooks } = useAppContext()
  const { getString } = useStrings()

  const ref = useRef<HTMLDivElement | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const containerKey = selectedItemData ? selectedItemData.id : 'default-key'
  useEffect(() => {
    // Clear the container first
    if (ref.current) {
      ref.current.innerHTML = ''
    }

    if (stepNameLogKeyMap && content) {
      content.split(/\r?\n/).forEach(line => {
        if (ref.current) {
          ref.current.appendChild(lineElement(line))
        }
      })
    }
  }, [content, stepNameLogKeyMap, selectedItemData])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const getLogData = (logBaseKey: string, status: string, onMessageStreaming: (e: any) => void) => {
    const logContent = hooks?.useLogsContent([logBaseKey])
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onError = (e: any) => {
      if (e.type === 'error') {
        streamingData?.closeStream()
      }
    }
    const streamingData = hooks?.useLogsStreaming([logBaseKey], onMessageStreaming, onError)

    if (
      selectedItemData?.status === StepStatus.RUNNING &&
      status !== capitalizeFirstLetter(StepStatus.SUCCESS) &&
      !isEmpty(logContent.streamData)
    ) {
      return streamingData.streamData[logBaseKey]
    }
    return logContent.blobDataCur
  }
  const [isBottom, setIsBottom] = useState(false)
  const [expandedStates, setExpandedStates] = useState<Map<string, { expanded: boolean; streaming: boolean }>>(
    new Map()
  )

  useEffect(() => {
    const states = new Map<string, { expanded: boolean; streaming: boolean }>()
    if (expandedStates.size === 0) {
      stepNameLogKeyMap?.forEach(_ => {
        states.set(_.logBaseKey, { expanded: false, streaming: false })
      })
      setExpandedStates(states)
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stepNameLogKeyMap])

  const handleClick = () => {
    const logContainer = containerRef.current as HTMLDivElement
    const scrollParent = logContainer?.parentElement as HTMLDivElement
    if (!isBottom) {
      scrollParent.scrollTop = scrollParent.scrollHeight
      setIsBottom(true)
    } else if (isBottom) {
      scrollParent.scrollTop = 0
      setIsBottom(false)
    }
  }
  // Function to toggle expanded state of a container
  const toggleExpandedState = useCallback((key: string) => {
    setExpandedStates(prevStates => {
      const newStates = new Map(prevStates)
      const currentState = newStates.get(key) || { expanded: false, streaming: false }
      newStates.set(key, { ...currentState, expanded: !currentState.expanded })
      return newStates
    })
  }, [])

  const [steps, setSteps] = useState(new Map())

  useEffect(() => {
    if (stepNameLogKeyMap) {
      const newStatuses = new Map()
      stepNameLogKeyMap.forEach((value, key) => {
        newStatuses.set(key, value)
      })
      setSteps(newStatuses)
    }
  }, [stepNameLogKeyMap, selectedItemData?.status, toggleExpandedState])

  const renderedSteps = useMemo(() => {
    return Array.from(steps?.entries() || []).map(([key, data], idx) => {
      if (key === undefined || idx === 0) {
        return
      }

      const expanded =
        expandedStates.get(data.logBaseKey)?.expanded || data.status === 'AsyncWaiting' || data.status === 'Queued'
      return (
        <Container ref={containerRef} key={data.logBaseKey} className={cx(css.pipelineSteps)}>
          <Container className={css.stepContainer}>
            <Layout.Horizontal
              spacing="small"
              className={cx(css.stepHeader, {
                [css.expanded]: expanded,
                [css.selected]: expanded
              })}
              {...ButtonRoleProps}
              onClick={() => {
                toggleExpandedState(data.logBaseKey)
              }}>
              <NavArrowRight color={Utils.getRealCSSColor(Color.GREY_500)} className={cx(css.noShrink, css.chevron)} />
              <ExecutionStatus
                className={cx(css.status, css.noShrink)}
                status={data.status.toLowerCase() as ExecutionState}
                iconSize={16}
                noBackground
                iconOnly
              />
              <Text className={css.name} lineClamp={1}>
                {key === StepTypes.LITEENGINETASK ? StepTypes.INITIALIZE : key}
              </Text>

              <FlexExpander />
              <Render when={data.timeStart && data.timeEnd}>
                <Text color={Color.GREY_300} font={{ variation: FontVariation.SMALL }} className={css.noShrink}>
                  {timeDistance(data.timeStart, data.timeEnd)}
                </Text>
              </Render>
            </Layout.Horizontal>
            <Render when={expanded}>
              <LogStageContainer
                value={key}
                status={data.status}
                getLogData={getLogData}
                logKey={data.logBaseKey}
                expanded={expanded}
                setSelectedStage={setSelectedStage}
              />
              <Button
                size={ButtonSize.SMALL}
                variation={ButtonVariation.PRIMARY}
                text={isBottom ? getString('top') : getString('bottom')}
                icon={isBottom ? 'arrow-up' : 'arrow-down'}
                iconProps={{ size: 10 }}
                onClick={handleClick}
                className={css.scrollDownBtn}
              />
            </Render>
          </Container>
        </Container>
      )
    }) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [steps, selectedItemData, toggleExpandedState, expandedStates, isBottom])
  return (
    <>
      {steps && (selectedItemData?.payload?.kind as EnumCheckPayloadKindExtended) === 'harness_stage' ? (
        renderedSteps
      ) : (
        <Container key={`nolog_${containerKey}`} ref={ref} className={cx(css.main, className)} />
      )}
    </>
  )
}

export const lineElement = (line = '') => {
  const element = document.createElement('pre')
  element.className = css.line

  // Function to escape special HTML characters
  const escapeHtml = (unsafe: string) => {
    return unsafe
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;')
  }

  // Escaping HTML special characters in the line
  const escapedLine = escapeHtml(line.replace(/\r?\n$/, ''))
  element.innerHTML = Anser.ansiToHtml(escapedLine)
  return element
}

export const LogViewer = React.memo(LogTerminal)

export interface LogStageContainerProps {
  stepNameLogKeyMap?: Map<string, string>
  expanded?: boolean

  getLogData: (logKey: string, status: string, onMessageStreaming: (e: any) => void) => any
  logKey: string
  status: string
  value: string
  setSelectedStage: React.Dispatch<React.SetStateAction<TypesStage | null>>
}

export const LogStageContainer: React.FC<LogStageContainerProps> = ({
  getLogData,
  expanded,
  logKey,
  value,
  status,
  setSelectedStage
}) => {
  const localRef = useRef<HTMLDivElement | null>(null) // Create a unique ref for each LogStageContainer instance
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const onMessageStreaming = (e: any) => {
    if (e.data) {
      sendStreamLogToRenderer(e.data || '')
    }
  }
  const data = getLogData(logKey, status, onMessageStreaming)
  const sendStreamLogToRenderer = useScheduleJob({
    handler: useCallback((blocks: string[]) => {
      const logContainer = localRef.current as HTMLDivElement

      if (logContainer) {
        const fragment = new DocumentFragment()

        blocks.forEach((block: string) => {
          const blockData = JSON.parse(block)
          const linePos = blockData.pos + 1
          const localDate = new Date(blockData.time)
          const formattedDate = localDate.toLocaleString()

          fragment.appendChild(
            lineElement(`${linePos}  ${blockData.level}  ${formattedDate.replace(',', '')}  ${blockData.out}`)
          )
        })

        const scrollParent = logContainer.parentElement?.parentElement?.parentElement as HTMLDivElement
        const autoScroll =
          scrollParent && scrollParent.scrollTop === scrollParent.scrollHeight - scrollParent.offsetHeight

        logContainer?.appendChild(fragment)

        if (autoScroll || scrollParent.scrollTop === 0) {
          scrollParent.scrollTop = scrollParent.scrollHeight
        }
      }
    }, []),
    isStreaming: true
  })
  useEffect(() => {
    if (expanded && status !== StepStatus.RUNNING) {
      const pipelineArr = parseLogString(data)
      const fragment = new DocumentFragment()
      // Clear the container first
      if (localRef.current) {
        localRef.current.innerHTML = ''
      }
      if (pipelineArr) {
        pipelineArr?.forEach((line: LogLine) => {
          const linePos = line.pos + 1
          const localDate = new Date(line.time)
          // Format date to a more readable format (local time)
          const formattedDate = localDate.toLocaleString()
          fragment.appendChild(
            lineElement(`${linePos}  ${line.logLevel}  ${formattedDate.replace(',', '')}  ${line.message}`)
          )
        })
        const logContainer = localRef.current as HTMLDivElement
        logContainer.appendChild(fragment)
      }
    } else if (status === StepStatus.RUNNING) {
      sendStreamLogToRenderer(data || '')
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expanded, data, setSelectedStage])
  return <Container key={`harnesslog_${value}`} ref={localRef} className={cx(css.main, css.stepLogContainer)} />
}
