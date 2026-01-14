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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { Container, Layout, Text } from '@harnessio/uicore'
import cx from 'classnames'
import { Falsy, Match, Truthy } from 'react-jsx-match'
import { useAppContext } from 'AppContext'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { useQueryParams } from 'hooks/useQueryParams'
import { useShowRequestError } from 'hooks/useShowRequestError'
import type { TypesExecution, TypesStage } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { ButtonRoleProps, PullRequestSection } from 'utils/Utils'
import { findDefaultExecution } from './ChecksUtils'
import css from './Checks.module.scss'

interface CheckPipelineStagesProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  pipelineName: string
  executionNumber: string
  expanded?: boolean
  onSelectStage: (stage: TypesStage) => void
}

export const CheckPipelineStages: React.FC<CheckPipelineStagesProps> = ({
  pipelineName,
  executionNumber,
  expanded,
  repoMetadata,
  pullReqMetadata,
  onSelectStage
}) => {
  const { data, error, loading, refetch } = useGet<TypesExecution>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${pipelineName}/executions/${executionNumber}`,
    lazy: true
  })
  const [execution, setExecution] = useState<TypesExecution>()
  const { uid, stageId } = useQueryParams<{ pullRequestId: string; uid: string; stageId: string }>()
  const stages = useMemo(() => execution?.stages || [], [execution])
  const history = useHistory()
  const { routes } = useAppContext()

  useShowRequestError(error, 0)

  useEffect(() => {
    let timeoutId = 0

    if (repoMetadata && expanded) {
      if (!execution && !error) {
        refetch()
      } else {
        if (
          !error &&
          stages.find(({ status }) => status === ExecutionState.PENDING || status === ExecutionState.RUNNING)
        ) {
          timeoutId = window.setTimeout(refetch, POLLING_INTERVAL)
        }
      }
    }

    return () => {
      window.clearTimeout(timeoutId)
    }
  }, [repoMetadata, expanded, execution, refetch, error, stages])
  const selectStage = useCallback(
    (stage: TypesStage) => {
      history.replace(
        routes.toCODEPullRequest({
          repoPath: repoMetadata.path as string,
          pullRequestId: String(pullReqMetadata.number),
          pullRequestSection: PullRequestSection.CHECKS
        }) + `?uid=${pipelineName}${`&stageId=${stage.name}`}`
      )
      onSelectStage(stage)
    },
    [history, onSelectStage, pipelineName, pullReqMetadata.number, repoMetadata.path, routes]
  )
  const stageRef = useRef<TypesStage>()

  useEffect(() => {
    if (data) {
      setExecution(data)
    }
  }, [data])

  useEffect(() => {
    if (!expanded) {
      setExecution(undefined)
    }
  }, [expanded])

  useEffect(() => {
    if (stages.length) {
      if (uid === pipelineName) {
        // Pre-select stage if no stage is selected in the url
        if (!stageId) {
          selectStage(findDefaultExecution(stages) as TypesStage)
        } else {
          // If a stage is selected in url, find if it's matched
          // with a stage from polling data and update selected
          // stage accordingly to make sure parents has the latest
          // stage data (status, time, etc...)
          const _stage = stages.find(stg => stg.name === stageId && stageRef.current !== stg)

          if (_stage) {
            stageRef.current = _stage
            selectStage(_stage)
          }
        }
      }
    }
  }, [selectStage, pipelineName, stageId, stages, uid])

  return (
    <Container className={cx(css.pipelineStages, { [css.hidden]: !expanded || error })}>
      <Match expr={loading && !execution}>
        <Truthy>
          <Container className={css.spinner}>
            <Icon name="steps-spinner" size={16} />
          </Container>
        </Truthy>
        <Falsy>
          <>
            {stages.map(stage => (
              <Layout.Horizontal
                spacing="small"
                key={stage.name}
                className={cx(css.subMenu, { [css.selected]: pipelineName === uid && stage.name === stageId })}
                {...ButtonRoleProps}
                onClick={() => {
                  // ALways send back the latest stage
                  selectStage(stages.find(stg => stg === stage.name) || stage)
                }}>
                <ExecutionStatus
                  className={cx(css.status, css.noShrink)}
                  status={stage.status as ExecutionState}
                  iconSize={16}
                  noBackground
                  iconOnly
                />
                <Text color={Color.GREY_800} className={css.text}>
                  {stage.name}
                </Text>
              </Layout.Horizontal>
            ))}
          </>
        </Falsy>
      </Match>
    </Container>
  )
}

const POLLING_INTERVAL = 2000
