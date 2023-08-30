import { Icon } from '@harnessio/icons'
import { FlexExpander, Layout } from '@harnessio/uicore'
import React, { FC, useEffect } from 'react'
import { useGet } from 'restful-react'
import { Text } from '@harnessio/uicore'
import type { TypesStep } from 'services/code'
import { timeDistance } from 'utils/Utils'
import ConsoleLogs from 'components/ConsoleLogs/ConsoleLogs'
import css from './ConsoleStep.module.scss'

interface ConsoleStepProps {
  step: TypesStep | undefined
  stageNumber: number | undefined
  spaceName: string
  pipelineName: string | undefined
  executionNumber: number
}

const ConsoleStep: FC<ConsoleStepProps> = ({ step, stageNumber, spaceName, pipelineName, executionNumber }) => {
  const [isOpened, setIsOpened] = React.useState(false)

  const { data, error, loading, refetch } = useGet<string>({
    path: `/api/v1/pipelines/${spaceName}/${pipelineName}/+/executions/${executionNumber}/logs/${String(
      stageNumber
    )}/${String(step?.number)}`,
    lazy: true
  })

  // this refetches any open steps when the stage number changes - really it shouldnt refetch until reopened...
  useEffect(() => {
    setIsOpened(false)
    refetch()
  }, [stageNumber, refetch])

  return (
    <>
      <Layout.Horizontal
        className={css.stepLayout}
        spacing="medium"
        onClick={() => {
          setIsOpened(!isOpened)
          if (!data && !loading) refetch()
        }}>
        <Icon name={isOpened ? 'chevron-down' : 'chevron-right'} />
        <Icon name={step?.status === 'Success' ? 'success-tick' : 'circle'} />
        <Text>{step?.name}</Text>
        <FlexExpander />
        {step?.started && step?.stopped && <div>{timeDistance(step?.stopped, step?.started)}</div>}
      </Layout.Horizontal>

      {isOpened ? (
        loading ? (
          <div className={css.loading}>Loading...</div>
        ) : error ? (
          <div>Error: {error}</div>
        ) : data ? (
          <ConsoleLogs logs={data} />
        ) : null
      ) : null}
    </>
  )
}

export default ConsoleStep
