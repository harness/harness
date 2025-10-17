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
import { Render } from 'react-jsx-match'
import { Container, Text, TableV2, Layout } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import type { TypesDefaultReviewerApprovalsResponse } from 'services/code'
import { PullReqReviewDecision } from '../PullRequestUtils'
import ReviewersPanel from '../Conversation/PullRequestOverviewPanel/sections/ReviewersPanel'
import css from '../PullRequest.module.scss'

interface DefaultReviewersPanelProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  defaultRevApprovalResponse: TypesDefaultReviewerApprovalsResponse[]
}

export const DefaultReviewersPanel: React.FC<DefaultReviewersPanelProps> = ({
  defaultRevApprovalResponse,
  pullReqMetadata
}) => {
  const { getString } = useStrings()

  const columns = useMemo(
    () =>
      [
        {
          id: 'REQUIRED',
          width: '36%',
          sort: true,
          Header: getString('required'),
          accessor: 'REQUIRED',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponse>) => {
            if (row.original?.minimum_required_count && row.original?.minimum_required_count > 0)
              return (
                <Text
                  lineClamp={1}
                  padding={{ left: 'small', right: 'small' }}
                  color={Color.BLACK}
                  flex={{ justifyContent: 'space-between' }}>
                  {row.original.current_count} / {row.original.minimum_required_count}
                </Text>
              )
            else if (row.original?.minimum_required_count_latest && row.original?.minimum_required_count_latest > 0)
              return (
                <Layout.Horizontal>
                  <Text
                    lineClamp={1}
                    padding={{ left: 'small', right: 'small' }}
                    color={Color.BLACK}
                    flex={{ justifyContent: 'space-between' }}>
                    {row.original.current_count} / {row.original.minimum_required_count_latest}
                  </Text>
                  <Text color={Color.GREY_350}>({getString('onLatestChanges')})</Text>
                </Layout.Horizontal>
              )
            else <Text>{row.original.current_count}</Text>
          }
        },
        {
          id: 'DefaultReviewers',
          width: '20%',
          sort: true,
          Header: getString('defaultReviewers'),
          accessor: 'DefaultReviewers',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponse>) => {
            return (
              <ReviewersPanel
                principals={row.original?.principals || []}
                userGroups={row.original?.user_groups || []}
              />
            )
          }
        },
        {
          id: 'changesRequested',
          Header: getString('changesRequestedBy'),
          width: '24%',
          sort: true,
          accessor: 'ChangesRequested',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponse>) => {
            const changeReqEvaluations = row.original?.evaluations
              ?.filter(evaluation => evaluation.decision === PullReqReviewDecision.CHANGEREQ)
              .map(evaluation => evaluation?.reviewer ?? {})

            return <ReviewersPanel principals={changeReqEvaluations || []} />
          }
        },
        {
          id: 'approvedBy',
          Header: getString('approvedBy'),
          sort: true,
          width: '20%',
          accessor: 'APPROVED BY',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponse>) => {
            const approvedEvaluations = row.original?.evaluations
              ?.filter(
                evaluation =>
                  evaluation.decision === PullReqReviewDecision.APPROVED &&
                  (row.original.minimum_required_count_latest ? evaluation.sha === pullReqMetadata?.source_sha : true)
              )
              .map(evaluation => evaluation?.reviewer || {})

            return <ReviewersPanel principals={approvedEvaluations || []} />
          }
        }
      ] as unknown as Column<TypesDefaultReviewerApprovalsResponse>[], // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  )
  return (
    <Render when={defaultRevApprovalResponse?.length}>
      <Container>
        <Layout.Vertical spacing="small">
          <TableV2
            className={css.reviewerTable}
            sortable
            columns={columns}
            data={defaultRevApprovalResponse as TypesDefaultReviewerApprovalsResponse[]}
            getRowClassName={() => css.row}
          />
        </Layout.Vertical>
      </Container>
    </Render>
  )
}
