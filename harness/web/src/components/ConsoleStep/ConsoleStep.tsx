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

import { Icon } from '@harnessio/icons'
import { Container, FlexExpander, Layout } from '@harnessio/uicore'
import React, { FC, useEffect, useRef, useState } from 'react'
import { useGet } from 'restful-react'
import { Text } from '@harnessio/uicore'
import type { LivelogLine, TypesStep } from 'services/code'
import { timeDistance } from 'utils/Utils'
import ConsoleLogs, { createStreamedLogLineElement } from 'components/ConsoleLogs/ConsoleLogs'
import { useStrings } from 'framework/strings'
import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import css from './ConsoleStep.module.scss'

interface ConsoleStepProps {
  step: TypesStep | undefined
  stageNumber: number | undefined
  repoPath: string
  pipelineName: string | undefined
  executionNumber: number
}

const ConsoleStep: FC<ConsoleStepProps> = ({ step, stageNumber, repoPath, pipelineName, executionNumber }) => {
  const { getString } = useStrings()

  const [isOpened, setIsOpened] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)
  const StreamingLogRef = useRef<HTMLDivElement | null>(null)

  const shouldUseGet = step?.status !== ExecutionState.RUNNING && step?.status !== ExecutionState.PENDING
  const isPending = step?.status === ExecutionState.PENDING

  const { data, error, loading, refetch } = useGet<LivelogLine[]>({
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipelineName}/executions/${executionNumber}/logs/${String(
      stageNumber
    )}/${String(step?.number)}`,
    lazy: true
  })

  useEffect(() => {
    setIsOpened(false)
  }, [stageNumber])

  useEffect(() => {
    if (step?.status === ExecutionState.RUNNING && isOpened) {
      setIsStreaming(true)
      if (StreamingLogRef.current) {
        StreamingLogRef.current.textContent = ''
      }

      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
      eventSourceRef.current = new EventSource(
        `/api/v1/repos/${repoPath}/+/pipelines/${pipelineName}/executions/${executionNumber}/logs/${String(
          stageNumber
        )}/${String(step?.number)}/stream`
      )
      eventSourceRef.current.onmessage = event => {
        const newLog = JSON.parse(event.data)
        const element = createStreamedLogLineElement(newLog)
        StreamingLogRef.current?.appendChild(element)
      }
    }
    return () => {
      setIsStreaming(false)
      if (step?.status === ExecutionState.RUNNING && isOpened) {
        refetch()
      }
      if (eventSourceRef.current) eventSourceRef.current.close()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [executionNumber, isOpened, pipelineName, repoPath, stageNumber, step?.name, step?.number, step?.status])

  let icon
  if (step?.status === ExecutionState.SUCCESS) {
    icon = <Icon name="success-tick" />
  } else if (isPending) {
    icon = <Icon name="circle" />
  } else if (step?.status === ExecutionState.RUNNING) {
    icon = <Icon className={css.spin} name="pending" />
  } else if (step?.status === ExecutionState.FAILURE) {
    icon = <Icon name="danger-icon" />
  } else if (step?.status === ExecutionState.SKIPPED) {
    icon = <Icon className={css.timeoutIcon} name="execution-timeout" />
  } else {
    icon = <Icon name="circle" /> // Default icon in case of other statuses or unknown status
  }

  let content
  if (!isOpened) {
    content = null
  } else if (loading) {
    content = <div className={css.loading}>{getString('loading')}</div>
  } else if (error && step?.status !== ExecutionState.RUNNING) {
    content = <div>Error: {error.message}</div>
  } else if (isStreaming) {
    content = <Container ref={StreamingLogRef} />
  } else if (data) {
    content = <ConsoleLogs logs={data} />
  }

  return (
    <>
      <Layout.Horizontal
        className={css.stepLayout}
        spacing="medium"
        onClick={() => {
          if (!isPending && step?.status !== ExecutionState.SKIPPED) {
            setIsOpened(!isOpened)
            if (shouldUseGet && !isOpened) refetch()
          }
        }}>
        <Icon name={isOpened ? 'chevron-down' : 'chevron-right'} />
        {/* TODO - flesh icon logic out */}
        {icon}
        <Text className={css.stepName}>{step?.name}</Text>
        <FlexExpander />
        {step?.started && step?.stopped && (
          <Text className={css.time}>{timeDistance(step?.stopped, step?.started, true)}</Text>
        )}
      </Layout.Horizontal>

      {isOpened && content}
    </>
  )
}

export default ConsoleStep
