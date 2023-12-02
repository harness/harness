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

import React, { useMemo } from 'react'
import { Falsy, Match, Render, Truthy } from 'react-jsx-match'
import { Container, Layout, Text, useToggle, Button, ButtonVariation, ButtonSize, Utils } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { Link } from 'react-router-dom'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { ExecutionStatus, ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import { useShowRequestError } from 'hooks/useShowRequestError'
import type { TypesCheck, TypesCodeOwnerEvaluation } from 'services/code'
import { useAppContext } from 'AppContext'
import { PullRequestCheckType, PullRequestSection, timeDistance } from 'utils/Utils'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import css from './ChecksOverview.module.scss'

interface ChecksOverviewProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  prChecksDecisionResult: PRChecksDecisionResult
  codeOwners?: TypesCodeOwnerEvaluation
}

export function ChecksOverview({
  repoMetadata,
  pullRequestMetadata,
  prChecksDecisionResult,
  codeOwners
}: ChecksOverviewProps) {
  const { getString } = useStrings()
  const [isExpanded, toggleExpanded] = useToggle(false)
  const { data, overallStatus, error, color, message } = prChecksDecisionResult

  useShowRequestError(error)

  return overallStatus && data?.length ? (
    <Container
      className={css.main}
      margin={{ bottom: pullRequestMetadata.description && !codeOwners ? undefined : 'large' }}
      style={{ '--border-color': Utils.getRealCSSColor(color) } as React.CSSProperties}>
      <Match expr={isExpanded}>
        <Truthy>
          <CheckSections repoMetadata={repoMetadata} pullRequestMetadata={pullRequestMetadata} data={data} />
        </Truthy>
        <Falsy>
          <Layout.Horizontal spacing="small" className={css.layout}>
            <ExecutionStatus status={overallStatus} noBackground iconOnly />
            <Text font={{ variation: FontVariation.LEAD }}>{getString('pr.checks')}</Text>
            <Text color={color} padding={{ left: 'small' }} font={{ variation: FontVariation.FORM_MESSAGE_WARNING }}>
              {message}
            </Text>
          </Layout.Horizontal>
        </Falsy>
      </Match>
      <Button
        className={css.showMore}
        variation={ButtonVariation.LINK}
        size={ButtonSize.SMALL}
        text={getString(isExpanded ? 'showLess' : 'showMore')}
        rightIcon={isExpanded ? 'main-chevron-up' : 'main-chevron-down'}
        iconProps={{ size: 10, margin: { left: 'xsmall' } }}
        onClick={toggleExpanded}
      />
    </Container>
  ) : null
}

interface CheckSectionsProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  data: TypesCheck[]
}

const CheckSections: React.FC<CheckSectionsProps> = ({ repoMetadata, pullRequestMetadata, data }) => {
  const [checks, pipelines] = useMemo(
    () =>
      data.reduce(
        ([_checks, _pipelines], item) => {
          if (item.payload?.kind === PullRequestCheckType.PIPELINE) {
            _pipelines.push(item)
          } else {
            _checks.push(item)
          }
          return [_checks, _pipelines]
        },
        [[], []] as [TypesCheck[], TypesCheck[]]
      ),
    [data]
  )

  return (
    <Container className={css.checks}>
      <Layout.Vertical spacing="medium">
        <CheckSection
          repoMetadata={repoMetadata}
          pullRequestMetadata={pullRequestMetadata}
          data={pipelines}
          isPipeline
        />
        <CheckSection repoMetadata={repoMetadata} pullRequestMetadata={pullRequestMetadata} data={checks} />
      </Layout.Vertical>
    </Container>
  )
}

const CheckSection: React.FC<CheckSectionsProps & { isPipeline?: boolean }> = ({
  repoMetadata,
  pullRequestMetadata,
  data,
  isPipeline
}) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()

  return (
    <Render when={data.length}>
      <Container>
        <Layout.Vertical spacing="small">
          <Text
            icon={isPipeline ? 'pipeline' : 'main-search'}
            iconProps={{ size: isPipeline ? 20 : 12, padding: { right: 'small' } }}
            font={{ variation: FontVariation.SMALL_BOLD }}>
            {getString(isPipeline ? 'pageTitle.pipelines' : 'checks')}
          </Text>
          <Container className={css.table}>
            {data.map(({ uid, status, summary, created, updated }) => (
              <Container className={css.row} key={uid}>
                <Layout.Horizontal className={css.rowLayout}>
                  <Container className={css.status}>
                    <ExecutionStatus status={status as ExecutionState} />
                  </Container>

                  <Link
                    to={
                      routes.toCODEPullRequest({
                        repoPath: repoMetadata.path as string,
                        pullRequestId: String(pullRequestMetadata.number),
                        pullRequestSection: PullRequestSection.CHECKS
                      }) + `?uid=${uid}`
                    }>
                    <Text font={{ variation: FontVariation.SMALL_BOLD }} className={css.name} lineClamp={1}>
                      {uid}
                    </Text>
                  </Link>

                  <Text className={css.desc} font={{ variation: FontVariation.SMALL }} lineClamp={1}>
                    {summary}
                  </Text>

                  <Text className={css.time} font={{ variation: FontVariation.SMALL }}>
                    {timeDistance(updated, created)}
                  </Text>
                </Layout.Horizontal>
              </Container>
            ))}
          </Container>
        </Layout.Vertical>
      </Container>
    </Render>
  )
}
