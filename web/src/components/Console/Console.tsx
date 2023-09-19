import React, { FC } from 'react'
import { useParams } from 'react-router-dom'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CODEProps } from 'RouteDefinitions'
import type { TypesStage } from 'services/code'
import ConsoleStep from 'components/ConsoleStep/ConsoleStep'
import { timeDistance } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import useLiveTimer from 'hooks/useLiveTimeHook'
import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import css from './Console.module.scss'

interface ConsoleProps {
  stage: TypesStage | undefined
  repoPath: string
}

const Console: FC<ConsoleProps> = ({ stage, repoPath }) => {
  const { pipeline, execution: executionNum } = useParams<CODEProps>()
  const { getString } = useStrings()
  const currentTime = useLiveTimer()

  return (
    <div className={css.container}>
      {stage?.error && (
        <Container className={css.error}>
          <Text font={{ variation: FontVariation.BODY }} color={Color.WHITE}>
            {stage?.error}
          </Text>
        </Container>
      )}
      <Container className={css.header}>
        <Layout.Horizontal className={css.headerLayout} spacing="small">
          <Text font={{ variation: FontVariation.H4 }} color={Color.WHITE} padding={{ left: 'large', right: 'large' }}>
            {stage?.name}
          </Text>
          {stage?.stopped && (
            <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
              {getString(
                stage.status === ExecutionState.KILLED ? 'executions.canceledTime' : 'executions.completedTime',
                { timeString: timeDistance(stage?.stopped, currentTime, true) }
              )}
            </Text>
          )}
        </Layout.Horizontal>
      </Container>
      <Layout.Vertical className={css.steps} spacing="small">
        {stage?.steps?.map((step, index) => (
          <ConsoleStep
            key={index}
            step={step}
            executionNumber={Number(executionNum)}
            pipelineName={pipeline}
            repoPath={repoPath}
            stageNumber={stage.number}
          />
        ))}
      </Layout.Vertical>
    </div>
  )
}

export default Console
