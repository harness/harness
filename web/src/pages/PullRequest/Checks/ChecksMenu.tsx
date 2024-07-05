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

import React, { useEffect, useMemo, useState } from 'react'
import { Render } from 'react-jsx-match'
import { NavArrowRight } from 'iconoir-react'
import { get, isEmpty, sortBy } from 'lodash-es'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { Container, Layout, Text, FlexExpander, Utils } from '@harnessio/uicore'
import ReactTimeago from 'react-timeago'
import { Color, FontVariation } from '@harnessio/design-system'
import {
  ButtonRoleProps,
  PullRequestCheckType,
  PullRequestSection,
  generateAlphaNumericHash,
  timeDistance
} from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useQueryParams } from 'hooks/useQueryParams'
import type { EnumCheckPayloadKind, TypesCheck, TypesStage } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { CheckPipelineStages } from './CheckPipelineStages'
import { ChecksProps, extractBetweenPipelinesAndExecutions, findDefaultExecution } from './ChecksUtils'
import css from './Checks.module.scss'

interface ChecksMenuProps extends ChecksProps {
  onDataItemChanged: (itemData: TypesCheck) => void
  setSelectedStage: (stage: TypesStage | null) => void
}

type TypesCheckPayloadExtended = EnumCheckPayloadKind | 'harness_stage'
type ExpandedStates = { [key: string]: boolean }
type ElapsedTimeStatusMap = { [key: string]: { status: 'string'; time: string; started: string } }

enum CheckKindPayload {
  HARNESS_STAGE = 'harness_stage'
}
export const ChecksMenu: React.FC<ChecksMenuProps> = ({
  repoMetadata,
  pullReqMetadata,
  prChecksDecisionResult,
  onDataItemChanged,
  setSelectedStage: setSelectedStageFromProps
}) => {
  const { getString } = useStrings()
  const { routes, standalone } = useAppContext()
  const history = useHistory()
  const { uid } = useQueryParams<{ uid: string }>()
  const [selectedUID, setSelectedUID] = React.useState<string | undefined>()
  const [selectedStage, setSelectedStage] = useState<TypesStage | null>(null)
  const checksData = useMemo(() => sortBy(prChecksDecisionResult?.data || [], ['uid']), [prChecksDecisionResult?.data])
  useMemo(() => {
    if (selectedUID) {
      const selectedDataItem = checksData.find(item => item.identifier === selectedUID)
      if (selectedDataItem) {
        onDataItemChanged(selectedDataItem)
      }
    }
  }, [selectedUID, checksData, onDataItemChanged])

  useEffect(() => {
    if (uid) {
      if (uid !== selectedUID && checksData.find(item => item.identifier === uid)) {
        setSelectedUID(uid)
      }
    } else {
      const defaultSelectedItem = findDefaultExecution(checksData)

      if (defaultSelectedItem) {
        onDataItemChanged(defaultSelectedItem)
        setSelectedUID(defaultSelectedItem.identifier)
        history.replace(
          routes.toCODEPullRequest({
            repoPath: repoMetadata.path as string,
            pullRequestId: String(pullReqMetadata.number),
            pullRequestSection: PullRequestSection.CHECKS
          }) + `?uid=${defaultSelectedItem.identifier}${selectedStage ? `&stageId=${selectedStage.name}` : ''}`
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
    pullReqMetadata.number,
    onDataItemChanged,
    selectedStage
  ])
  const [expandedPipelineId, setExpandedPipelineId] = useState<string | null>(null)
  const [statusTimeStates, setStatusTimeStates] = useState<{
    [key: string]: { status: string; time: string; started: string }
  }>({})

  const groupByPipeline = (data: TypesCheck[]) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return data.reduce((acc: any, item: any) => {
      const hash = generateAlphaNumericHash(6)
      const rawPipelineName = !isEmpty(item.link) ? extractBetweenPipelinesAndExecutions(item.link) : ''
      const rawPipelineId = rawPipelineName !== '' ? rawPipelineName : `raw-${hash}`

      const pipelineId =
        (item?.payload?.kind as TypesCheckPayloadExtended) === CheckKindPayload.HARNESS_STAGE
          ? item?.payload?.data?.pipeline_identifier
          : rawPipelineId

      if (!acc[pipelineId]) {
        acc[pipelineId] = []
      }
      acc[pipelineId].push(item)

      return acc
    }, {})
  }
  const toggleExpandedState = (key: string) => {
    setExpandedPipelineId(prevKey => (prevKey === key ? null : key))
  }

  const groupedData = useMemo(() => groupByPipeline(checksData), [checksData])
  useEffect(() => {
    const initialStates: ExpandedStates = {}
    const initialMap: ElapsedTimeStatusMap = {}
    Object.keys(groupedData).forEach(key => {
      const findStatus = () => {
        const statusPriority = ['running', 'pending', 'failure', 'error', 'success']

        for (const status of statusPriority) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const foundObject = groupedData[key].find((obj: any) => obj.status === status)
          if (foundObject) return foundObject
        }

        return null
      }

      const dataArr = groupedData[key]
      if (groupedData && dataArr) {
        const { minCreated, maxUpdated } = dataArr.reduce(
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          (acc: any, item: TypesCheck) => ({
            minCreated: item.started && item.started < acc.minCreated ? item.started : acc.minCreated,
            maxUpdated: item.ended && item.ended > acc.maxUpdated ? item.ended : acc.maxUpdated
          }),
          { minCreated: Infinity, maxUpdated: -Infinity }
        )
        const res = findStatus()
        const statusVal = res ? res.status : ''
        initialMap[key] = {
          status: statusVal,
          time: timeDistance(minCreated, maxUpdated),
          started: groupedData[key][0].started
        }
      }
      if (uid) {
        if (uid.includes(key)) {
          setExpandedPipelineId(key)
        }
        initialStates[key] = uid.includes(key)
      } else {
        setExpandedPipelineId(null)
        initialStates[key] = false
      }
    })
    setStatusTimeStates(initialMap)
  }, [groupedData, uid])

  const customFormatter = (_value: number, _unit: string, _suffix: string, date: Date | string | number) => {
    const now = new Date()
    const then = new Date(date)
    const secondsPast = (now.getTime() - then.getTime()) / 1000
    const days = Math.round(secondsPast / 86400)
    const remainder = secondsPast % 86400
    const hours = Math.floor(remainder / 3600)
    const minutes = Math.floor((remainder % 3600) / 60)
    const seconds = Math.floor(remainder % 60)

    return getString('customTime', {
      days: days ? getString('customDay', { days }) : '',
      hours: hours ? getString('customHour', { hours }) : '',
      minutes: minutes ? getString('customMin', { minutes }) : '',
      seconds: seconds ? getString('customSecond', { seconds }) : ''
    })
  }
  return (
    <Layout.Vertical padding={{ top: 'large' }} spacing={'small'} className={cx(css.menu, css.leftPaneContent)}>
      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      {Object.entries(groupedData).map(([pipelineId, checks]: any) => (
        <Container
          key={pipelineId}
          onClick={() => {
            toggleExpandedState(pipelineId)
          }}
          className={cx(css.leftPaneMenuItem, {
            [css.expanded]: expandedPipelineId === pipelineId && !pipelineId.includes('raw-') && !standalone,
            [css.layout]: !pipelineId.includes('raw-') && !standalone,
            [css.menuItem]: !pipelineId.includes('raw-') && !standalone,
            [css.hideStages]: expandedPipelineId !== pipelineId && !pipelineId.includes('raw-') && !standalone
          })}>
          {!standalone && !pipelineId.includes('raw-') && (
            <Layout.Horizontal className={css.layout}>
              <Render when={statusTimeStates[pipelineId]?.status}>
                <ExecutionStatus
                  className={cx(css.status, css.noShrink)}
                  status={statusTimeStates[pipelineId]?.status as ExecutionState}
                  iconSize={22}
                  noBackground
                  iconOnly
                />
              </Render>
              <Text className={css.uid} lineClamp={1} padding={{ left: 'small' }}>
                {pipelineId}
              </Text>
              <FlexExpander />

              <NavArrowRight
                color={Utils.getRealCSSColor(Color.GREY_500)}
                className={cx(css.noShrink, css.chevron)}
                strokeWidth="1.5"
              />
            </Layout.Horizontal>
          )}
          {(checks as TypesCheck[]).map((itemData: TypesCheck) => (
            <Container key={`container_${itemData.identifier}`} className={css.checkMenuItemContainer}>
              <CheckMenuItem
                repoMetadata={repoMetadata}
                pullReqMetadata={pullReqMetadata}
                prChecksDecisionResult={prChecksDecisionResult}
                key={itemData.identifier}
                itemData={itemData}
                customFormatter={customFormatter}
                isPipeline={itemData.payload?.kind === PullRequestCheckType.PIPELINE}
                isSelected={itemData.identifier === selectedUID}
                onClick={stage => {
                  setSelectedUID(itemData.identifier)
                  setSelectedStage(stage || null)
                  setSelectedStageFromProps(stage || null)

                  history.replace(
                    routes.toCODEPullRequest({
                      repoPath: repoMetadata.path as string,
                      pullRequestId: String(pullReqMetadata.number),
                      pullRequestSection: PullRequestSection.CHECKS
                    }) + `?uid=${itemData.identifier}${stage ? `&stageId=${stage.name}` : ''}`
                  )
                }}
                setSelectedStage={stage => {
                  setSelectedStage(stage)
                  setSelectedStageFromProps(stage)
                }}
              />
            </Container>
          ))}
        </Container>
      ))}
    </Layout.Vertical>
  )
}

interface CheckMenuItemProps extends ChecksProps {
  isPipeline?: boolean
  isSelected?: boolean
  itemData: TypesCheck
  onClick: (stage?: TypesStage) => void
  setSelectedStage: (stage: TypesStage | null) => void
  customFormatter: (_value: number, _unit: string, _suffix: string, date: Date | string | number) => string
}

const CheckMenuItem: React.FC<CheckMenuItemProps> = ({
  isPipeline,
  isSelected = false,
  itemData,
  onClick,
  repoMetadata,
  pullReqMetadata,
  setSelectedStage,
  customFormatter
}) => {
  const [expanded, setExpanded] = useState(isSelected)

  useEffect(() => {
    if (isSelected) {
      setExpanded(isSelected)
    }
  }, [isSelected])
  const name =
    itemData?.identifier &&
    itemData?.identifier.includes('-') &&
    (itemData.payload?.kind as TypesCheckPayloadExtended) === CheckKindPayload.HARNESS_STAGE
      ? itemData.identifier.split('-')[1]
      : itemData.identifier
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
        onClick={e => {
          e.stopPropagation()
          if (isPipeline) {
            setExpanded(!expanded)
          } else {
            onClick()
          }
        }}>
        <ExecutionStatus
          className={cx(css.status, css.noShrink)}
          status={itemData.status as ExecutionState}
          iconSize={18}
          noBackground
          iconOnly
        />
        <Text className={css.uid} lineClamp={1}>
          {name}
        </Text>

        <FlexExpander />

        <Text color={Color.GREY_300} font={{ variation: FontVariation.SMALL }} className={css.noShrink}>
          {itemData?.ended && itemData?.started ? (
            timeDistance(itemData.started, itemData.ended)
          ) : (
            <ReactTimeago date={new Date(itemData?.started || 0)} formatter={customFormatter} />
          )}
        </Text>

        <Render when={isPipeline}>
          <NavArrowRight
            color={Utils.getRealCSSColor(Color.GREY_500)}
            className={cx(css.noShrink, css.chevron)}
            strokeWidth="1.5"
          />
        </Render>
      </Layout.Horizontal>

      <Render when={isPipeline}>
        <CheckPipelineStages
          pipelineName={itemData.identifier as string}
          executionNumber={get(itemData, 'payload.data.execution_number', '')}
          expanded={expanded}
          repoMetadata={repoMetadata}
          pullReqMetadata={pullReqMetadata}
          onSelectStage={setSelectedStage}
        />
      </Render>
    </Container>
  )
}
