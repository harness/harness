import React, { FC } from 'react'
import { useParams } from 'react-router-dom'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { TypesStage } from 'services/code'
import ConsoleStep from 'components/ConsoleStep/ConsoleStep'
import { timeDistance } from 'utils/Utils'
import css from './Console.module.scss'

interface ConsoleProps {
  stage: TypesStage | undefined
}

const Console: FC<ConsoleProps> = ({ stage }) => {
  const space = useGetSpaceParam()
  const { pipeline, execution: executionNum } = useParams<CODEProps>()

  return (
    <div className={css.container}>
      <Container className={css.header}>
        <Layout.Horizontal className={css.headerLayout} spacing="small">
          <Text font={{ variation: FontVariation.H4 }} color={Color.WHITE} padding={{ left: 'large', right: 'large' }}>
            {stage?.name}
          </Text>
          {stage?.started && stage?.stopped && (
            <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
              {/* this needs fixed */}
              {timeDistance(stage?.started, stage?.stopped)}
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
            spaceName={space}
            stageNumber={stage.number}
          />
        ))}
      </Layout.Vertical>
    </div>
  )
}

export default Console
