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

import React, { useState } from 'react'
import { PopoverInteractionKind, Spinner } from '@blueprintjs/core'
import { useGet, useMutate } from 'restful-react'
import { isEmpty, omit } from 'lodash-es'
import cx from 'classnames'
import { Container, Layout, Text, Avatar, FlexExpander, useToaster, Utils, stringSubstitute } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useAppContext } from 'AppContext'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useStrings } from 'framework/strings'
import type {
  TypesPullReq,
  RepoRepositoryOutput,
  TypesScopesLabels,
  PullreqCombinedListResponse,
  EnumPullReqReviewDecision
} from 'services/code'
import { ColorName, getErrorMessage, PrincipalType } from 'utils/Utils'
import { ReviewerSelect } from 'components/ReviewerSelect/ReviewerSelect'
import {
  PullReqReviewDecision,
  generateReviewDecisionInfo,
  processReviewDecision
} from 'pages/PullRequest/PullRequestUtils'
import { LabelSelector } from 'components/Label/LabelSelector/LabelSelector'
import { Label } from 'components/Label/Label'
import { getConfig } from 'services/config'
import ignoreFailed from '../../../../icons/ignoreFailed.svg?url'
import css from './PullRequestSideBar.module.scss'

interface PullRequestSideBarProps {
  combinedReviewers?: PullreqCombinedListResponse | null
  labels: TypesScopesLabels | null
  repoMetadata: RepoRepositoryOutput
  pullRequestMetadata: TypesPullReq
  refetchReviewers: () => void
  refetchLabels: () => void
  refetchActivities: () => void
}

const PullRequestSideBar = (props: PullRequestSideBarProps) => {
  const [labelQuery, setLabelQuery] = useState<string>('')
  const { standalone } = useAppContext()
  const {
    combinedReviewers,
    repoMetadata,
    pullRequestMetadata,
    refetchReviewers,
    labels,
    refetchLabels,
    refetchActivities
  } = props
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()

  const { reviewers, user_group_reviewers } = combinedReviewers || {}

  const { mutate: addReviewer } = useMutate({
    verb: 'PUT',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/reviewers`
  })
  const { mutate: addUserGroupReviewer } = useMutate({
    verb: 'PUT',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/reviewers/usergroups`
  })

  const { mutate: removeReviewer } = useMutate({
    verb: 'DELETE',
    path: ({ id }) => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviewers/${id}`
  })
  const { mutate: removeUserGroupReviewer } = useMutate({
    verb: 'DELETE',
    path: ({ id }) =>
      `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviewers/usergroups/${id}`
  })

  const { mutate: removeLabel, loading: removingLabel } = useMutate({
    verb: 'DELETE',
    base: getConfig('code/api/v1'),
    path: ({ label_id }) => `/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/labels/${label_id}`
  })

  const {
    data: labelsList,
    refetch: refetchLabelsList,
    loading: labelListLoading
  } = useGet<TypesScopesLabels>({
    base: getConfig('code/api/v1'),
    path: `/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/labels`,
    queryParams: { assignable: true, query: labelQuery },
    debounce: 500
  })

  //TODO: add actions when you click the options menu button and also api integration when there's optional and required reviwers
  return (
    <Container width={`30%`}>
      <Container padding={{ left: 'xxlarge' }}>
        <Layout.Vertical>
          <Layout.Horizontal>
            <Text style={{ lineHeight: '24px' }} font={{ variation: FontVariation.H6 }}>
              {getString('reviewers')}
            </Text>
            <FlexExpander />
            <ReviewerSelect
              pullRequestMetadata={pullRequestMetadata}
              standalone={standalone}
              onSelect={async normalizedPrincipal => {
                try {
                  const { id, type } = normalizedPrincipal
                  if (type === PrincipalType.USER) {
                    await addReviewer({ reviewer_id: id })
                  } else {
                    await addUserGroupReviewer({ usergroup_id: id })
                  }
                  refetchActivities?.()
                  refetchReviewers?.()
                } catch (err) {
                  showError(getErrorMessage(err))
                }
              }}
            />
          </Layout.Horizontal>
          <Container padding={{ top: 'medium', bottom: 'large' }}>
            {!isEmpty(user_group_reviewers) &&
              user_group_reviewers?.map(reviewer => {
                const { decision = PullReqReviewDecision.PENDING, user_group, sha } = reviewer
                const updatedReviewDecision = processReviewDecision(decision, sha, pullRequestMetadata?.source_sha)
                const areChangesRequested = decision === PullReqReviewDecision.CHANGEREQ
                const reviewDecisionInfo = generateReviewDecisionInfo(updatedReviewDecision)

                return (
                  <Container key={`user_group-${user_group?.id}`} className={css.alignLayout}>
                    <Utils.WrapOptionalTooltip
                      tooltip={
                        <Text color={Color.GREY_100} padding="small">
                          {reviewDecisionInfo.message}
                        </Text>
                      }
                      tooltipProps={{ isDark: true, interactionKind: PopoverInteractionKind.HOVER }}>
                      {updatedReviewDecision === PullReqReviewDecision.OUTDATED ? (
                        <img className={css.svgOutdated} src={ignoreFailed} width={20} height={20} />
                      ) : (
                        <Icon
                          className={areChangesRequested ? css.redIcon : undefined}
                          {...omit(reviewDecisionInfo, 'iconProps')}
                        />
                      )}
                    </Utils.WrapOptionalTooltip>
                    <Icon
                      margin={'xsmall'}
                      className={cx(css.reviewerAvatar, css.ugicon)}
                      name="user-groups"
                      size={18}
                    />
                    <Text lineClamp={1} style={{ paddingLeft: '3px' }} className={css.reviewerName}>
                      {user_group?.name}
                    </Text>
                    <FlexExpander />
                    <OptionsMenuButton
                      isDark={true}
                      icon="Options"
                      iconProps={{ size: 14 }}
                      style={{ paddingBottom: '9px' }}
                      width="100px"
                      height="24px"
                      items={[
                        {
                          isDanger: true,
                          text: getString('remove'),
                          onClick: () => {
                            removeUserGroupReviewer({}, { pathParams: { id: user_group?.id } })
                              .then(() => refetchActivities())
                              .catch(err => {
                                showError(getErrorMessage(err))
                              })
                            if (refetchReviewers) {
                              refetchReviewers?.()
                            }
                          }
                        }
                      ]}
                    />
                  </Container>
                )
              })}
            {!isEmpty(reviewers) &&
              reviewers?.map(reviewer => {
                const { review_decision, sha, reviewer: principal } = reviewer
                const updatedReviewDecision = processReviewDecision(
                  review_decision as EnumPullReqReviewDecision,
                  sha,
                  pullRequestMetadata?.source_sha
                )
                const areChangesRequested = updatedReviewDecision === PullReqReviewDecision.CHANGEREQ
                const reviewDecisionInfo = generateReviewDecisionInfo(updatedReviewDecision)

                return (
                  <Layout.Horizontal key={`principal-${principal?.id}`} className={css.alignLayout}>
                    <Utils.WrapOptionalTooltip
                      tooltip={
                        <Text color={Color.GREY_100} padding="small">
                          {reviewDecisionInfo.message}
                        </Text>
                      }
                      tooltipProps={{ isDark: true, interactionKind: PopoverInteractionKind.HOVER }}>
                      {updatedReviewDecision === PullReqReviewDecision.OUTDATED ? (
                        <img className={css.svgOutdated} src={ignoreFailed} width={20} height={20} />
                      ) : (
                        <Icon
                          className={areChangesRequested ? css.redIcon : undefined}
                          {...omit(reviewDecisionInfo, 'iconProps')}
                        />
                      )}
                    </Utils.WrapOptionalTooltip>
                    <Avatar
                      className={cx(css.reviewerAvatar, {
                        [css.iconPadding]: !areChangesRequested
                      })}
                      name={principal?.display_name}
                      size="small"
                      hoverCard={false}
                    />

                    <Text lineClamp={1} className={css.reviewerName}>
                      {principal?.display_name}
                    </Text>
                    <FlexExpander />
                    <OptionsMenuButton
                      isDark={true}
                      icon="Options"
                      iconProps={{ size: 14 }}
                      style={{ paddingBottom: '9px' }}
                      width="100px"
                      height="24px"
                      items={[
                        {
                          isDanger: true,
                          text: getString('remove'),
                          onClick: () => {
                            removeReviewer({}, { pathParams: { id: principal?.id } })
                              .then(() => refetchActivities())
                              .catch(err => {
                                showError(getErrorMessage(err))
                              })
                            if (refetchReviewers) {
                              refetchReviewers?.()
                            }
                          }
                        }
                      ]}
                    />
                  </Layout.Horizontal>
                )
              })}
            {isEmpty(reviewers) && isEmpty(user_group_reviewers) && (
              <Text color={Color.GREY_300} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                {getString('noReviewers')}
              </Text>
            )}
          </Container>
        </Layout.Vertical>

        <Layout.Vertical>
          <Layout.Horizontal>
            <Text style={{ lineHeight: '24px' }} font={{ variation: FontVariation.H6 }}>
              {getString('labels.labels')}
            </Text>
            <FlexExpander />

            <LabelSelector
              pullRequestMetadata={pullRequestMetadata}
              allLabelsData={labelsList}
              refetchLabels={refetchLabels}
              refetchlabelsList={refetchLabelsList}
              repoMetadata={repoMetadata}
              query={labelQuery}
              setQuery={setLabelQuery}
              labelListLoading={labelListLoading}
              refetchActivities={refetchActivities}
            />
          </Layout.Horizontal>
          <Container padding={{ top: 'medium', bottom: 'large' }}>
            <Layout.Horizontal className={css.labelsLayout}>
              {!isEmpty(labels?.label_data) ? (
                labels?.label_data?.map((label, index) => (
                  <Label
                    key={index}
                    name={label.key as string}
                    label_color={label.color as ColorName}
                    label_value={{
                      name: label.assigned_value?.value as string,
                      color: label.assigned_value?.color as ColorName
                    }}
                    scope={label.scope}
                    removeLabelBtn={true}
                    disableRemoveBtnTooltip={true}
                    handleRemoveClick={() => {
                      removeLabel({}, { pathParams: { label_id: label.id } })
                        .then(() => {
                          label.assigned_value?.value
                            ? showSuccess(
                                stringSubstitute(getString('labels.removedLabel'), {
                                  label: `${label.key}:${label.assigned_value?.value}`
                                }) as string
                              )
                            : showSuccess(
                                stringSubstitute(getString('labels.removedLabel'), {
                                  label: label.key
                                }) as string
                              )
                          refetchActivities()
                        })
                        .catch(err => {
                          showError(getErrorMessage(err))
                        })
                      refetchLabels()
                    }}
                  />
                ))
              ) : (
                <Text color={Color.GREY_300} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                  {getString('labels.noLabels')}
                </Text>
              )}
              {removingLabel && <Spinner size={16} />}
            </Layout.Horizontal>
          </Container>
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

export default PullRequestSideBar
