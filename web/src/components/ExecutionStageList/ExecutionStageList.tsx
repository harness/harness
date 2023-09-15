import React, { FC } from 'react'
import { Container, FlexExpander, Layout, Text } from '@harnessio/uicore'
import cx from 'classnames'
import type { TypesStage } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/ExecutionUtils'
import { timeDistance } from 'utils/Utils'
import useLiveTimer from 'hooks/useLiveTimeHook'
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
  const isActive = stage?.status === ExecutionState.RUNNING
  const currentTime = useLiveTimer(isActive)

  return (
    <Container
      className={css.menuItem}
      onClick={() => {
        setSelectedStage(stage.number || null)
      }}>
      <Layout.Horizontal spacing="small" className={cx(css.layout, { [css.selected]: isSelected })}>
        <ExecutionStatus
          status={getStatus(stage.status || ExecutionState.PENDING)}
          iconOnly
          noBackground
          iconSize={18}
          className={css.statusIcon}
          isCi
        />
        <Text className={css.uid} lineClamp={1}>
          {stage.name}
        </Text>
        <FlexExpander />
        {stage.started && (stage.stopped || isActive) && (
          <Text style={{ fontSize: '12px' }}>
            {/* Use live time when running, static time when finished */}
            {timeDistance(stage.started, isActive ? currentTime : stage.stopped, true)}
          </Text>
        )}
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
