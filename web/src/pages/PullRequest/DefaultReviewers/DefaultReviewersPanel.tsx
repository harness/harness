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
import { Container, Text, TableV2, Layout, Avatar } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import type { TypesDefaultReviewerApprovalsResponseWithRevDecision } from 'utils/Utils'
import { PullReqReviewDecision } from '../PullRequestUtils'
import css from '../CodeOwners/CodeOwnersOverview.module.scss'

interface DefaultReviewersPanelProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  defaultRevApprovalResponse: TypesDefaultReviewerApprovalsResponseWithRevDecision[]
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
          width: '40%',
          sort: true,
          Header: getString('required'),
          accessor: 'REQUIRED',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponseWithRevDecision>) => {
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
          width: '18%',
          sort: true,
          Header: getString('defaultReviewers'),
          accessor: 'DefaultReviewers',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponseWithRevDecision>) => {
            return (
              <Layout.Horizontal
                key={`keyContainer-${row.original.rule_info?.identifier}`}
                className={css.ownerContainer}
                spacing="tiny">
                {row.original.principals?.map((principal, idx) => {
                  if (idx < 2) {
                    return (
                      <Avatar
                        key={`text-${principal?.display_name}-${idx}-avatar`}
                        hoverCard={true}
                        email={principal?.email || ' '}
                        size="small"
                        name={principal?.display_name || ''}
                      />
                    )
                  }
                  if (idx === 2 && row.original.principals?.length && row.original.principals?.length > 2) {
                    return (
                      <Text
                        key={`text-${principal?.display_name}-${idx}-top`}
                        padding={{ top: 'xsmall' }}
                        tooltipProps={{ isDark: true }}
                        tooltip={
                          <Container width={215} padding={'small'}>
                            <Layout.Horizontal key={`tooltip-${idx}`} className={css.ownerTooltip}>
                              {row.original.principals?.map((entry, entryidx) => (
                                <Text
                                  key={`text-${entry?.display_name}-${entryidx}`}
                                  lineClamp={1}
                                  color={Color.GREY_0}
                                  padding={{ right: 'small' }}>
                                  {row.original.principals?.length === entryidx + 1
                                    ? `${entry?.display_name}`
                                    : `${entry?.display_name}, `}
                                </Text>
                              ))}
                            </Layout.Horizontal>
                          </Container>
                        }
                        flex={{ alignItems: 'center' }}>{`+${row.original.principals?.length - 2}`}</Text>
                    )
                  }
                  return null
                })}
              </Layout.Horizontal>
            )
          }
        },
        {
          id: 'changesRequested',
          Header: getString('changesRequestedBy'),
          width: '24%',
          sort: true,
          accessor: 'ChangesRequested',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponseWithRevDecision>) => {
            const changeReqEvaluations = row?.original?.principals?.filter(
              principal => principal?.review_decision === PullReqReviewDecision.CHANGEREQ
            )
            return (
              <Layout.Horizontal className={css.ownerContainer} spacing="tiny">
                {changeReqEvaluations?.map((principal, idx: number) => {
                  if (idx < 2) {
                    return (
                      <Avatar
                        key={`approved-${principal?.display_name}-avatar`}
                        hoverCard={true}
                        email={principal?.email || ' '}
                        size="small"
                        name={principal?.display_name || ''}
                      />
                    )
                  }
                  if (idx === 2 && changeReqEvaluations.length && changeReqEvaluations.length > 2) {
                    return (
                      <Text
                        key={`approved-${principal?.display_name}-text`}
                        padding={{ top: 'xsmall' }}
                        tooltipProps={{ isDark: true }}
                        tooltip={
                          <Container width={215} padding={'small'}>
                            <Layout.Horizontal className={css.ownerTooltip}>
                              {changeReqEvaluations?.map(evalPrincipal => (
                                <Text
                                  key={`approved-${evalPrincipal?.display_name}`}
                                  lineClamp={1}
                                  color={Color.GREY_0}
                                  padding={{ right: 'small' }}>{`${evalPrincipal?.display_name}, `}</Text>
                              ))}
                            </Layout.Horizontal>
                          </Container>
                        }
                        flex={{ alignItems: 'center' }}>{`+${changeReqEvaluations.length - 2}`}</Text>
                    )
                  }
                  return null
                })}
              </Layout.Horizontal>
            )
          }
        },
        {
          id: 'approvedBy',
          Header: getString('approvedBy'),
          sort: true,
          width: '15%',
          accessor: 'APPROVED BY',
          Cell: ({ row }: CellProps<TypesDefaultReviewerApprovalsResponseWithRevDecision>) => {
            const approvedEvaluations = row?.original?.principals?.filter(
              principal =>
                principal.review_decision === PullReqReviewDecision.APPROVED &&
                (row.original.minimum_required_count_latest
                  ? principal.review_sha === pullReqMetadata?.source_sha
                  : true)
            )

            return (
              <Layout.Horizontal className={css.ownerContainer} spacing="tiny">
                {approvedEvaluations?.map((principal, idx: number) => {
                  if (idx < 2) {
                    return (
                      <Avatar
                        key={`approved-${principal?.display_name}-avatar`}
                        hoverCard={true}
                        email={principal?.email || ' '}
                        size="small"
                        name={principal?.display_name || ''}
                      />
                    )
                  }
                  if (idx === 2 && approvedEvaluations.length && approvedEvaluations.length > 2) {
                    return (
                      <Text
                        key={`approved-${principal?.display_name}-text`}
                        padding={{ top: 'xsmall' }}
                        tooltipProps={{ isDark: true }}
                        tooltip={
                          <Container width={215} padding={'small'}>
                            <Layout.Horizontal className={css.ownerTooltip}>
                              {approvedEvaluations?.map(appPrincipalObj => (
                                <Text
                                  key={`approved-${appPrincipalObj?.display_name}`}
                                  lineClamp={1}
                                  color={Color.GREY_0}
                                  padding={{ right: 'small' }}>{`${appPrincipalObj?.display_name}, `}</Text>
                              ))}
                            </Layout.Horizontal>
                          </Container>
                        }
                        flex={{ alignItems: 'center' }}>{`+${approvedEvaluations.length - 2}`}</Text>
                    )
                  }
                  return null
                })}
              </Layout.Horizontal>
            )
          }
        }
      ] as unknown as Column<TypesDefaultReviewerApprovalsResponseWithRevDecision>[], // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  )
  return (
    <Render when={defaultRevApprovalResponse?.length}>
      <Container>
        <Layout.Vertical spacing="small">
          <TableV2
            className={css.codeOwnerTable}
            sortable
            columns={columns}
            data={defaultRevApprovalResponse as TypesDefaultReviewerApprovalsResponseWithRevDecision[]}
            getRowClassName={() => css.row}
          />
        </Layout.Vertical>
      </Container>
    </Render>
  )
}
