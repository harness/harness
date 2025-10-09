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

import React, { FC } from 'react'
import { Container, FlexExpander, Layout, Text } from '@harnessio/uicore'
import cx from 'classnames'
import type { TypesStage } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/ExecutionUtils'
import { timeDifferenceInMinutesAndSeconds } from 'utils/Utils'
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
            {timeDifferenceInMinutesAndSeconds(stage.started, isActive ? currentTime : stage.stopped)}
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
