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
import { Render } from 'react-jsx-match'
import { NavArrowRight } from 'iconoir-react'
import cx from 'classnames'
import { useGet } from 'restful-react'
import Anser from 'anser'
import DOMPurify from 'dompurify'
import { Container, Layout, Text, FlexExpander, Utils, useToaster } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { ButtonRoleProps, getErrorMessage, timeDistance } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import type { LivelogLine, TypesStage, TypesStep } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { useScheduleJob } from 'hooks/useScheduleJob'
import { useShowRequestError } from 'hooks/useShowRequestError'
import css from './Checks.module.scss'

interface CheckPipelineStepsProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  pipelineName: string
  executionNumber: string
  stage: TypesStage
}

export const CheckPipelineSteps: React.FC<CheckPipelineStepsProps> = ({
  repoMetadata,
  pullReqMetadata,
  pipelineName,
  stage,
  executionNumber
}) => {
  return (
    <Container className={cx(css.pipelineSteps)}>
      {stage.steps?.map(step => (
        <CheckPipelineStep
          key={pipelineName + stage.name + executionNumber + step.name + step.started}
          pipelineName={pipelineName}
          executionNumber={executionNumber}
          repoMetadata={repoMetadata}
          pullReqMetadata={pullReqMetadata}
          stage={stage}
          step={step}
        />
      ))}
    </Container>
  )
}

const CheckPipelineStep: React.FC<CheckPipelineStepsProps & { step: TypesStep }> = ({
  pipelineName,
  executionNumber,
  stage,
  repoMetadata,
  step
}) => {
  const { showError } = useToaster()
  const eventSourceRef = useRef<EventSource | null>(null)
  const isRunning = useMemo(() => step.status === ExecutionState.RUNNING, [step])
  const [expanded, setExpanded] = useState(
    isRunning || step.status === ExecutionState.ERROR || step.status === ExecutionState.FAILURE
  )
  const path = useMemo(
    () =>
      `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${pipelineName}/executions/${executionNumber}/logs/${stage.number}/${step.number}`,
    [executionNumber, pipelineName, repoMetadata?.path, stage.number, step.number]
  )
  const lazy =
    !expanded || isRunning || step.status === ExecutionState.PENDING || step.status === ExecutionState.SKIPPED
  const { data, error, loading, refetch } = useGet<LivelogLine[]>({
    path,
    lazy: true
  })
  const logs = useMemo(() => data?.map(({ out = '' }) => out), [data])
  const [isStreamingDone, setIsStreamingDone] = useState(false)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [autoCollapse, setAutoCollapse] = useState(false)
  const closeEventStream = useCallback((event?: Event) => {
    eventSourceRef.current?.close()
    eventSourceRef.current = null

    // Report to console an error if last event is not `eof`
    if (event) {
      if ((event as unknown as { data: string }).data !== 'eof') {
        console.error('An error has occurred while streaming through EventSource', event) // eslint-disable-line no-console
      }
    }
  }, [])
  const sendStreamLogToRenderer = useScheduleJob({
    handler: useCallback((blocks: string[]) => {
      const logContainer = containerRef.current as HTMLDivElement

      if (logContainer) {
        const fragment = new DocumentFragment()

        blocks.forEach(block => fragment.appendChild(createLogLineElement(block)))

        const scrollParent = logContainer.closest(`.${css.content}`) as HTMLDivElement
        const autoScroll =
          scrollParent && scrollParent.scrollTop === scrollParent.scrollHeight - scrollParent.offsetHeight

        logContainer?.appendChild(fragment)

        if (autoScroll) {
          scrollParent.scrollTop = scrollParent.scrollHeight
        }
      }
    }, []),
    isStreaming: true
  })
  const sendCompleteLogsToRenderer = useScheduleJob({
    handler: useCallback((blocks: string[]) => {
      const logContainer = containerRef.current as HTMLDivElement

      if (logContainer) {
        const fragment = new DocumentFragment()
        blocks.forEach(block => fragment.appendChild(createLogLineElement(block)))
        logContainer.appendChild(fragment)
      }
    }, []),
    maxProcessingBlockSize: 100
  })

  useEffect(
    function initStreamingLogs() {
      if (expanded && isRunning) {
        setAutoCollapse(false)

        if (containerRef.current) {
          containerRef.current.textContent = ''
        }

        eventSourceRef.current = new EventSource(`${path}/stream`)
        eventSourceRef.current.onmessage = event => {
          try {
            sendStreamLogToRenderer((JSON.parse(event.data) as LivelogLine).out || '')
          } catch (exception) {
            showError(getErrorMessage(exception))
            closeEventStream()
          }
        }

        eventSourceRef.current.onerror = event => {
          setIsStreamingDone(true)
          setAutoCollapse(true)
          closeEventStream(event)
        }
      } else {
        closeEventStream()
      }

      return closeEventStream
    },
    [expanded, isRunning, showError, path, step.status, closeEventStream, sendStreamLogToRenderer]
  )

  useEffect(
    function fetchAndRenderCompleteLogs() {
      if (!lazy && !error && !isRunning && !isStreamingDone && expanded) {
        if (!logs) {
          refetch()
        } else if (Array.isArray(logs)) {
          sendCompleteLogsToRenderer(logs)
        }
      }
    },
    [lazy, error, logs, refetch, isStreamingDone, expanded, isRunning, sendCompleteLogsToRenderer]
  )

  useEffect(
    function autoCollapseWhenStreamingIsDone() {
      if (autoCollapse && expanded && step.status === ExecutionState.SUCCESS) {
        setIsStreamingDone(false)
        setAutoCollapse(false)
        setExpanded(false)
      }
    },
    [autoCollapse, expanded, step.status]
  )

  useShowRequestError(error, 0)

  return (
    <Container key={step.number} className={css.stepContainer}>
      <Layout.Horizontal
        spacing="small"
        className={cx(css.stepHeader, { [css.expanded]: expanded, [css.selected]: expanded })}
        {...ButtonRoleProps}
        onClick={() => {
          setExpanded(!expanded)
        }}>
        <NavArrowRight color={Utils.getRealCSSColor(Color.GREY_500)} className={cx(css.noShrink, css.chevron)} />

        <ExecutionStatus
          className={cx(css.status, css.noShrink)}
          status={step.status as ExecutionState}
          iconSize={16}
          noBackground
          iconOnly
        />

        <Text className={css.name} lineClamp={1}>
          {step.name}
        </Text>

        <FlexExpander />

        <Render when={loading}>
          <Icon name="steps-spinner" size={16} />
        </Render>

        <Render when={step.started && step.stopped}>
          <Text color={Color.GREY_300} font={{ variation: FontVariation.SMALL }} className={css.noShrink}>
            {timeDistance(step.started, step.stopped)}
          </Text>
        </Render>
      </Layout.Horizontal>
      <Render when={expanded}>
        <Container className={css.stepLogContainer} ref={containerRef}></Container>
      </Render>
    </Container>
  )
}

export const createLogLineElement = (line = '') => {
  const element = document.createElement('pre') as HTMLPreElement & {
    setHTML: (html: string, options: Record<string, unknown>) => void
  }
  element.className = css.consoleLine

  const html = Anser.ansiToHtml(line.replace(/\r?\n$/, ''))

  if (window.Sanitizer && element.setHTML) {
    element.setHTML(html, {
      sanitizer: new window.Sanitizer({
        allowElements: ['span'],
        allowAttributes: { style: ['span'] }
      })
    })
  } else {
    element.innerHTML = DOMPurify.sanitize(html, {
      USE_PROFILES: { html: true },
      ALLOWED_TAGS: ['span'],
      ALLOWED_ATTR: ['style']
    })
  }

  return element
}
