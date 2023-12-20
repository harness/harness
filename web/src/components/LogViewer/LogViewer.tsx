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

import React, { useEffect, useRef, useState } from 'react'
import Anser from 'anser'
import cx from 'classnames'
import { Container, FlexExpander, Layout, Text, Utils } from '@harnessio/uicore'
import { NavArrowRight } from 'iconoir-react'
import { Color, FontVariation } from '@harnessio/design-system'
import { Render } from 'react-jsx-match'
import { parseLogString } from 'pages/PullRequest/Checks/ChecksUtils'
import { useAppContext } from 'AppContext'
import { ButtonRoleProps, timeDistance } from 'utils/Utils'
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

export type EnumCheckPayloadKindExtended = EnumCheckPayloadKind | 'harness_stage'

const LogTerminal: React.FC<LogViewerProps> = ({
  content,
  className,
  stepNameLogKeyMap,
  setSelectedStage,
  selectedItemData
}) => {
  const { hooks } = useAppContext()
  const ref = useRef<HTMLDivElement | null>(null)

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

  const getLogData = (logBaseKey: string) => {
    const logContent = hooks?.useLogsContentHook([logBaseKey])

    return logContent.blobDataCur
  }
  // State to manage expanded states of all containers

  const [expandedStates, setExpandedStates] = useState(new Map<string, boolean>())
  // UseEffect to initialize the states
  useEffect(() => {
    const states = new Map<string, boolean>()
    stepNameLogKeyMap?.forEach(value => {
      states.set(value.logBaseKey, false)
    })
    setExpandedStates(states) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  // Function to toggle expanded state of a container
  const toggleExpandedState = (key: string) => {
    setExpandedStates(prevStates => {
      const newStates = new Map(prevStates)
      newStates.set(key, !newStates.get(key))
      return newStates
    })
  }

  if (
    stepNameLogKeyMap &&
    (selectedItemData?.payload?.kind as EnumCheckPayloadKindExtended) === 'harness_stage' &&
    selectedItemData?.status !== 'running'
  ) {
    const renderedSteps = Array.from(stepNameLogKeyMap?.entries() || []).map(([key, data], idx) => {
      if (key === undefined || idx === 0) {
        return
      }
      return (
        <Container key={data.logBaseKey} className={cx(css.pipelineSteps)}>
          <Container className={css.stepContainer}>
            <Layout.Horizontal
              spacing="small"
              className={cx(css.stepHeader, {
                [css.expanded]: expandedStates.get(key),
                [css.selected]: expandedStates.get(key)
              })}
              {...ButtonRoleProps}
              onClick={() => {
                toggleExpandedState(key)
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
            <Render when={expandedStates.get(key)}>
              <LogStageContainer
                value={key}
                getLogData={getLogData}
                logKey={data.logBaseKey}
                expanded={expandedStates.get(key)}
                setSelectedStage={setSelectedStage}
              />
            </Render>
          </Container>
        </Container>
      )
    })
    return <>{renderedSteps}</>
  }
  return <Container key={`nolog_${containerKey}`} ref={ref} className={cx(css.main, className)} />
}

const lineElement = (line = '') => {
  const element = document.createElement('pre')
  element.className = css.line
  element.innerHTML = Anser.ansiToHtml(line.replace(/\r?\n$/, ''))
  return element
}

export const LogViewer = React.memo(LogTerminal)

export interface LogStageContainerProps {
  stepNameLogKeyMap?: Map<string, string>
  expanded?: boolean
  getLogData: (logKey: string) => any // eslint-disable-next-line @typescript-eslint/no-explicit-any
  logKey: string
  value: string
  setSelectedStage: React.Dispatch<React.SetStateAction<TypesStage | null>>
}

export const LogStageContainer: React.FC<LogStageContainerProps> = ({
  getLogData,
  stepNameLogKeyMap,
  expanded,
  logKey,
  value,
  setSelectedStage
}) => {
  const localRef = useRef<HTMLDivElement | null>(null) // Create a unique ref for each LogStageContainer instance
  const data = getLogData(logKey)

  useEffect(() => {
    if (expanded) {
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
    }
  }, [expanded, data, setSelectedStage, stepNameLogKeyMap])
  return <Container key={`harnesslog_${value}`} ref={localRef} className={cx(css.main, css.stepLogContainer)} />
}
