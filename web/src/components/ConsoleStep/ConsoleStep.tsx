import { Icon } from '@harnessio/icons'
import { FlexExpander, Layout } from '@harnessio/uicore'
import React, { FC, useEffect, useRef, useState } from 'react'
import { useGet } from 'restful-react'
import { Text } from '@harnessio/uicore'
import type { LivelogLine, TypesStep } from 'services/code'
import { timeDistance } from 'utils/Utils'
import ConsoleLogs from 'components/ConsoleLogs/ConsoleLogs'
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
  const [streamingLogs, setStreamingLogs] = useState<LivelogLine[]>([])
  const eventSourceRef = useRef<EventSource | null>(null)

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
    if (step?.status === ExecutionState.RUNNING) {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        setStreamingLogs([])
      }
      eventSourceRef.current = new EventSource(
        `/api/v1/repos/${repoPath}/+/pipelines/${pipelineName}/executions/${executionNumber}/logs/${String(
          stageNumber
        )}/${String(step?.number)}/stream`
      )
      eventSourceRef.current.onmessage = event => {
        const newLog = JSON.parse(event.data)
        setStreamingLogs(existingLogs => {
          return [...existingLogs, newLog]
        })
      }
    }
    return () => {
      if (eventSourceRef.current) eventSourceRef.current.close()
    }
  }, [executionNumber, pipelineName, repoPath, stageNumber, step?.number, step?.status])

  let icon
  if (step?.status === ExecutionState.SUCCESS) {
    icon = <Icon name="success-tick" />
  } else if (isPending) {
    icon = <Icon name="circle" />
  } else if (step?.status === ExecutionState.RUNNING) {
    icon = <Icon className={css.spin} name="pending" />
  } else {
    icon = <Icon name="circle" /> // Default icon in case of other statuses or unknown status
  }

  let content
  if (loading) {
    content = <div className={css.loading}>{getString('loading')}</div>
  } else if (error && step?.status !== ExecutionState.RUNNING) {
    content = <div>Error: {error.message}</div>
  } else if (streamingLogs.length) {
    content = <ConsoleLogs logs={streamingLogs} />
  } else if (data) {
    content = <ConsoleLogs logs={data} />
  }

  return (
    <>
      <Layout.Horizontal
        className={css.stepLayout}
        spacing="medium"
        onClick={() => {
          if (!isPending) {
            setIsOpened(!isOpened)
            if (shouldUseGet && !isOpened) refetch()
          }
        }}>
        <Icon name={isOpened ? 'chevron-down' : 'chevron-right'} />
        {/* TODO - flesh icon logic out */}
        {icon}
        <Text>{step?.name}</Text>
        <FlexExpander />
        {step?.started && step?.stopped && <div>{timeDistance(step?.stopped, step?.started)}</div>}
      </Layout.Horizontal>

      {isOpened && content}
    </>
  )
}

export default ConsoleStep
