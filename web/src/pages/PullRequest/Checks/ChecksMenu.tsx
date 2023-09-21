/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect, useMemo, useState } from 'react'
import { Render } from 'react-jsx-match'
import { NavArrowRight } from 'iconoir-react'
import { get, sortBy } from 'lodash-es'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { Container, Layout, Text, FlexExpander, Utils } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { ButtonRoleProps, PullRequestCheckType, PullRequestSection, timeDistance } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useQueryParams } from 'hooks/useQueryParams'
import type { TypesCheck, TypesStage } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { CheckPipelineStages } from './CheckPipelineStages'
import { ChecksProps, findDefaultExecution } from './ChecksUtils'
import css from './Checks.module.scss'

interface ChecksMenuProps extends ChecksProps {
  onDataItemChanged: (itemData: TypesCheck) => void
  setSelectedStage: (stage: TypesStage | null) => void
}

export const ChecksMenu: React.FC<ChecksMenuProps> = ({
  repoMetadata,
  pullRequestMetadata,
  prChecksDecisionResult,
  onDataItemChanged,
  setSelectedStage: setSelectedStageFromProps
}) => {
  const { routes } = useAppContext()
  const history = useHistory()
  const { uid } = useQueryParams<{ uid: string }>()
  const [selectedUID, setSelectedUID] = React.useState<string | undefined>()
  const [selectedStage, setSelectedStage] = useState<TypesStage | null>(null)
  const checksData = useMemo(() => sortBy(prChecksDecisionResult?.data || [], ['uid']), [prChecksDecisionResult?.data])

  useMemo(() => {
    if (selectedUID) {
      const selectedDataItem = checksData.find(item => item.uid === selectedUID)
      if (selectedDataItem) {
        onDataItemChanged(selectedDataItem)
      }
    }
  }, [selectedUID, checksData, onDataItemChanged])

  useEffect(() => {
    if (uid) {
      if (uid !== selectedUID && checksData.find(item => item.uid === uid)) {
        setSelectedUID(uid)
      }
    } else {
      const defaultSelectedItem = findDefaultExecution(checksData)

      if (defaultSelectedItem) {
        onDataItemChanged(defaultSelectedItem)
        setSelectedUID(defaultSelectedItem.uid)
        history.replace(
          routes.toCODEPullRequest({
            repoPath: repoMetadata.path as string,
            pullRequestId: String(pullRequestMetadata.number),
            pullRequestSection: PullRequestSection.CHECKS
          }) + `?uid=${defaultSelectedItem.uid}${selectedStage ? `&stageId=${selectedStage.name}` : ''}`
        )
      }
    }
  }, [
    uid,
    checksData,
    selectedUID,
    history,
    routes,
    repoMetadata.path,
    pullRequestMetadata.number,
    onDataItemChanged,
    selectedStage
  ])

  return (
    <Container className={css.menu}>
      {checksData.map(itemData => (
        <CheckMenuItem
          repoMetadata={repoMetadata}
          pullRequestMetadata={pullRequestMetadata}
          prChecksDecisionResult={prChecksDecisionResult}
          key={itemData.uid}
          itemData={itemData}
          isPipeline={itemData.payload?.kind === PullRequestCheckType.PIPELINE}
          isSelected={itemData.uid === selectedUID}
          onClick={stage => {
            setSelectedUID(itemData.uid)
            setSelectedStage(stage || null)
            setSelectedStageFromProps(stage || null)

            history.replace(
              routes.toCODEPullRequest({
                repoPath: repoMetadata.path as string,
                pullRequestId: String(pullRequestMetadata.number),
                pullRequestSection: PullRequestSection.CHECKS
              }) + `?uid=${itemData.uid}${stage ? `&stageId=${stage.name}` : ''}`
            )
          }}
          setSelectedStage={stage => {
            setSelectedStage(stage)
            setSelectedStageFromProps(stage)
          }}
        />
      ))}
    </Container>
  )
}

interface CheckMenuItemProps extends ChecksProps {
  isPipeline?: boolean
  isSelected?: boolean
  itemData: TypesCheck
  onClick: (stage?: TypesStage) => void
  setSelectedStage: (stage: TypesStage | null) => void
}

const CheckMenuItem: React.FC<CheckMenuItemProps> = ({
  isPipeline,
  isSelected = false,
  itemData,
  onClick,
  repoMetadata,
  pullRequestMetadata,
  setSelectedStage
}) => {
  const [expanded, setExpanded] = useState(isSelected)

  useEffect(() => {
    if (isSelected) {
      setExpanded(isSelected)
    }
  }, [isSelected])

  return (
    <Container className={css.menuItem}>
      <Layout.Horizontal
        spacing="small"
        className={cx(css.layout, {
          [css.expanded]: expanded,
          [css.selected]: isSelected,
          [css.forPipeline]: isPipeline
        })}
        {...ButtonRoleProps}
        onClick={() => {
          if (isPipeline) {
            setExpanded(!expanded)
          } else {
            onClick()
          }
        }}>
        <Render when={isPipeline}>
          <NavArrowRight
            color={Utils.getRealCSSColor(Color.GREY_500)}
            className={cx(css.noShrink, css.chevron)}
            strokeWidth="1.5"
          />
        </Render>

        <Text className={css.uid} lineClamp={1}>
          {itemData.uid}
        </Text>

        <FlexExpander />

        <Text color={Color.GREY_300} font={{ variation: FontVariation.SMALL }} className={css.noShrink}>
          {timeDistance(itemData.updated, itemData.created)}
        </Text>

        <ExecutionStatus
          className={cx(css.status, css.noShrink)}
          status={itemData.status as ExecutionState}
          iconSize={16}
          noBackground
          iconOnly
        />
      </Layout.Horizontal>

      <Render when={isPipeline}>
        <CheckPipelineStages
          pipelineName={itemData.uid as string}
          executionNumber={get(itemData, 'payload.data.execution_number', '')}
          expanded={expanded}
          repoMetadata={repoMetadata}
          pullRequestMetadata={pullRequestMetadata}
          onSelectStage={setSelectedStage}
        />
      </Render>
    </Container>
  )
}
