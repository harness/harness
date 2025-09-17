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
import {
  Container,
  Text,
  useToggle,
  Button,
  ButtonVariation,
  ButtonSize,
  Utils,
  TableV2,
  Layout
} from '@harnessio/uicore'
import cx from 'classnames'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { useShowRequestError } from 'hooks/useShowRequestError'
import type { TypesCodeOwnerEvaluation, TypesCodeOwnerEvaluationEntry, TypesUserGroupInfo } from 'services/code'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import { CodeOwnerReqDecision, UNKNOWN_GROUP } from 'utils/Utils'
import {
  PullReqReviewDecision,
  checkEntries,
  getCombinedEvaluations,
  findReviewDecisions,
  findWaitingDecisions
} from '../PullRequestUtils'
import ReviewersPanel from '../Conversation/PullRequestOverviewPanel/sections/ReviewersPanel'
import css from './CodeOwnersOverview.module.scss'
import prCss from '../PullRequest.module.scss'

interface ChecksOverviewProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  prChecksDecisionResult: PRChecksDecisionResult
  reqCodeOwnerLatestApproval: boolean
  codeOwners?: TypesCodeOwnerEvaluation
  standalone: boolean
}

export function CodeOwnersOverview({
  codeOwners,
  repoMetadata,
  pullReqMetadata,
  prChecksDecisionResult,
  reqCodeOwnerLatestApproval,
  standalone
}: ChecksOverviewProps) {
  const { getString } = useStrings()
  const [isExpanded, toggleExpanded] = useToggle(false)
  const { error } = prChecksDecisionResult

  useShowRequestError(error)

  const changeReqEntries = findReviewDecisions(codeOwners?.evaluation_entries, CodeOwnerReqDecision.CHANGEREQ)
  const approvalEntries = findReviewDecisions(codeOwners?.evaluation_entries, CodeOwnerReqDecision.APPROVED)
  const waitingEntries = findWaitingDecisions(
    pullReqMetadata,
    reqCodeOwnerLatestApproval,
    codeOwners?.evaluation_entries
  )

  const { borderColor, message, overallStatus } = checkEntries(
    getString,
    changeReqEntries,
    waitingEntries,
    approvalEntries,
    codeOwners?.evaluation_entries?.length || 0
  )
  return codeOwners?.evaluation_entries?.length ? (
    <Container
      className={cx(css.main, { [css.codeOwner]: !standalone })}
      margin={{ top: 'medium', bottom: pullReqMetadata?.description ? undefined : 'large' }}
      style={{ '--border-color': Utils.getRealCSSColor(borderColor) } as React.CSSProperties}>
      <Match expr={isExpanded}>
        <Truthy>
          {codeOwners && (
            <CodeOwnerSections repoMetadata={repoMetadata} pullReqMetadata={pullReqMetadata} data={codeOwners} />
          )}
        </Truthy>
        <Falsy>
          <Layout.Horizontal spacing="small" className={css.layout}>
            <ExecutionStatus inExecution={true} isCi={true} status={overallStatus} noBackground iconOnly />
            <Text font={{ variation: FontVariation.LEAD }}>{getString('codeOwner.title')}</Text>
            <Text
              color={borderColor}
              padding={{ left: 'small' }}
              font={{ variation: FontVariation.FORM_MESSAGE_WARNING }}>
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

interface CodeOwnerSectionsProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  data: TypesCodeOwnerEvaluation
  reqCodeOwnerLatestApproval?: boolean
}

const CodeOwnerSections: React.FC<CodeOwnerSectionsProps> = ({ repoMetadata, pullReqMetadata, data }) => {
  return (
    <Container className={css.checks}>
      <Layout.Vertical spacing="medium">
        <CodeOwnerSection repoMetadata={repoMetadata} pullReqMetadata={pullReqMetadata} data={data} />
      </Layout.Vertical>
    </Container>
  )
}

export const CodeOwnerSection: React.FC<CodeOwnerSectionsProps> = ({
  data,
  pullReqMetadata,
  reqCodeOwnerLatestApproval
}) => {
  const { getString } = useStrings()

  const columns = useMemo(
    () =>
      [
        {
          id: 'CODE',
          width: '36%',
          Header: getString('code'),
          sort: true,
          accessor: 'CODE',
          Cell: ({ row }: CellProps<TypesCodeOwnerEvaluationEntry>) => {
            return (
              <Text
                lineClamp={1}
                padding={{ left: 'small', right: 'small' }}
                color={Color.BLACK}
                flex={{ justifyContent: 'space-between' }}>
                {row.original.pattern}
              </Text>
            )
          }
        },
        {
          id: 'Owners',
          width: '20%',
          Header: getString('ownersHeading'),
          accessor: 'OWNERS',
          Cell: ({ row }: CellProps<TypesCodeOwnerEvaluationEntry>) => (
            <ReviewersPanel
              principals={(row.original.owner_evaluations || []).map(evaluation => evaluation?.owner || {})}
              userGroups={
                row.original.user_group_owner_evaluations?.map(
                  group =>
                    ({
                      identifier: group?.id || '',
                      name: group?.name || group?.id || UNKNOWN_GROUP
                    } as TypesUserGroupInfo)
                ) || []
              }
            />
          )
        },
        {
          id: 'changesRequested',
          Header: getString('changesRequestedBy'),
          width: '24%',
          accessor: 'ChangesRequested',
          Cell: ({ row }: CellProps<TypesCodeOwnerEvaluationEntry>) => {
            const changeReqEvaluations = getCombinedEvaluations(row?.original)
              ?.filter(evaluation => evaluation.review_decision === PullReqReviewDecision.CHANGEREQ)
              .map(evaluation => evaluation?.owner || {})

            return <ReviewersPanel principals={changeReqEvaluations || []} />
          }
        },
        {
          id: 'approvedBy',
          Header: getString('approvedBy'),
          width: '20%',
          accessor: 'APPROVED BY',
          Cell: ({ row }: CellProps<TypesCodeOwnerEvaluationEntry>) => {
            const approvedEvaluations = getCombinedEvaluations(row?.original)
              ?.filter(
                evaluation =>
                  evaluation.review_decision === PullReqReviewDecision.APPROVED &&
                  (reqCodeOwnerLatestApproval ? evaluation.review_sha === pullReqMetadata?.source_sha : true)
              )
              .map(evaluation => evaluation?.owner || {})

            return <ReviewersPanel principals={approvedEvaluations || []} />
          }
        }
      ] as Column<TypesCodeOwnerEvaluationEntry>[],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  )
  return (
    <Render when={data?.evaluation_entries?.length}>
      <Container>
        <Layout.Vertical spacing="small">
          <TableV2
            className={prCss.reviewerTable}
            sortable
            columns={columns}
            data={data?.evaluation_entries as TypesCodeOwnerEvaluationEntry[]}
            getRowClassName={() => prCss.row}
          />
        </Layout.Vertical>
      </Container>
    </Render>
  )
}

export default CodeOwnersOverview
