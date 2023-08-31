import React, { FC } from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import cx from 'classnames'
import type { TypesStage } from 'services/code'
import css from './ExecutionStageList.module.scss'

interface ExecutionStageListProps {
  stages: TypesStage[]
  selectedStage: number | null
  setSelectedStage: (selectedStep: number | null) => void
}

interface ExecutionStageProps {
  stage: TypesStage
  isSelected?: boolean
  selectedStage: number | null
  setSelectedStage: (selectedStage: number | null) => void
}

const ExecutionStage: FC<ExecutionStageProps> = ({ stage, isSelected = false, setSelectedStage }) => {
  return (
    <Container
      className={css.menuItem}
      onClick={() => {
        setSelectedStage(stage.number || null)
      }}>
      <Layout.Horizontal spacing="small" className={cx(css.layout, { [css.selected]: isSelected })}>
        <Icon name="success-tick" size={16} />
        <Text className={css.uid} lineClamp={1}>
          {stage.name}
        </Text>
      </Layout.Horizontal>
    </Container>
  )
}

const ExecutionStageList: FC<ExecutionStageListProps> = ({ stages, setSelectedStage, selectedStage }) => {
  return (
    <Container className={css.menu}>
      {stages.map((stage, index) => {
        return (
          <ExecutionStage
            key={index}
            stage={stage}
            isSelected={selectedStage === stage.number}
            selectedStage={selectedStage}
            setSelectedStage={setSelectedStage}
          />
        )
      })}
    </Container>
  )
}

export default ExecutionStageList
