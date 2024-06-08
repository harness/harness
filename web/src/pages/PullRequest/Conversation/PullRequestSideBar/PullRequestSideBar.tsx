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

import React from 'react'
import { PopoverInteractionKind } from '@blueprintjs/core'
import { useMutate } from 'restful-react'
import { omit } from 'lodash-es'
import cx from 'classnames'
import { Container, Layout, Text, Avatar, FlexExpander, useToaster, Utils } from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useStrings } from 'framework/strings'
import type { TypesPullReq, TypesRepository, EnumPullReqReviewDecision } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { ReviewerSelect } from 'components/ReviewerSelect/ReviewerSelect'
import { PullReqReviewDecision, processReviewDecision } from 'pages/PullRequest/PullRequestUtils'
import ignoreFailed from '../../../../icons/ignoreFailed.svg?url'
import css from './PullRequestSideBar.module.scss'

interface PullRequestSideBarProps {
  reviewers?: Unknown
  repoMetadata: TypesRepository
  pullRequestMetadata: TypesPullReq
  refetchReviewers: () => void
}

const PullRequestSideBar = (props: PullRequestSideBarProps) => {
  const { reviewers, repoMetadata, pullRequestMetadata, refetchReviewers } = props
  // const [searchTerm, setSearchTerm] = useState('')
  // const [page] = usePageIndex(1)
  const { getString } = useStrings()
  // const tagArr = []
  const { showError } = useToaster()

  const generateReviewDecisionInfo = (
    reviewDecision: EnumPullReqReviewDecision | PullReqReviewDecision.outdated
  ): {
    name: IconName
    color?: Color
    size?: number
    icon: IconName
    className?: string
    iconProps?: { color?: Color }
    message: string
  } => {
    let info: {
      name: IconName
      color?: Color
      size?: number
      className?: string
      icon: IconName
      iconProps?: { color?: Color }
      message: string
    }

    switch (reviewDecision) {
      case PullReqReviewDecision.changeReq:
        info = {
          name: 'error-transparent-no-outline',
          color: Color.RED_700,
          size: 18,
          className: css.redIcon,
          icon: 'error-transparent-no-outline',
          iconProps: { color: Color.RED_700 },
          message: 'requested changes'
        }
        break
      case PullReqReviewDecision.approved:
        info = {
          name: 'tick-circle',
          color: Color.GREEN_700,
          size: 16,
          icon: 'tick-circle',
          iconProps: { color: Color.GREEN_700 },
          message: 'approved changes'
        }
        break
      case PullReqReviewDecision.pending:
        info = {
          name: 'waiting',
          color: Color.GREY_700,
          size: 16,
          icon: 'waiting',
          iconProps: { color: Color.GREY_700 },
          message: 'pending review'
        }
        break
      case PullReqReviewDecision.outdated:
        info = {
          name: 'dot',
          color: Color.GREY_100,
          size: 16,
          icon: 'dot',
          message: 'outdated approval'
        }
        break
      default:
        info = {
          name: 'dot',
          color: Color.GREY_100,
          size: 16,
          icon: 'dot',
          message: 'status'
        }
    }

    return info
  }

  // const { data: reviewersData,refetch:refetchReviewers } = useGet<Unknown[]>({
  //   path: `/api/v1/principals`,
  //   queryParams: {
  //     query: searchTerm,
  //     limit: LIST_FETCHING_LIMIT,
  //     page: page,
  //     accountIdentifier: 'kmpySmUISimoRrJL6NL73w',
  //     type: 'user'
  //   }
  // })
  // console.log(reviewers)

  // interface PrReviewOption {
  //   method: 'optional' | 'required' | 'reject'
  //   title: string
  //   disabled?: boolean
  //   color: Color
  // }
  // const prDecisionOptions: PrReviewOption[] = [
  //   {
  //     method: 'optional',
  //     title: getString('optional'),
  //     color: Color.GREEN_700
  //   },
  //   {
  //     method: 'required',
  //     title: getString('required'),
  //     color: Color.ORANGE_700
  //   }
  // ]
  const { mutate: updateCodeCommentStatus } = useMutate({
    verb: 'PUT',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/reviewers`
  })
  const { mutate: removeReviewer } = useMutate({
    verb: 'DELETE',
    path: ({ id }) => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviewers/${id}`
  })
  // const [isOptionsOpen, setOptionsOpen] = React.useState(false)
  // const [val, setVal] = useState<SelectOption>()
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

            {/* <Popover
              isOpen={isOptionsOpen}
              onInteraction={nextOpenState => {
                setOptionsOpen(nextOpenState)
              }}
              content={
                <Menu>
                  {prDecisionOptions.map(option => {
                    return (
                      <Menu.Item
                        key={option.method}
                        // className={css.menuReviewItem}
                        disabled={false || option.disabled}
                        text={
                          <Layout.Horizontal>
                            <Text flex width={'fit-content'} font={{ variation: FontVariation.BODY }}>
                              {option.title}
                            </Text>
                          </Layout.Horizontal>
                        }
                        onClick={() => {
                          // setDecisionOption(option)
                        }}
                      />
                    )
                  })}
                </Menu>
              }
              usePortal={true}
              minimal={true}
              fill={false}
              position={Position.BOTTOM_RIGHT}> */}
            <ReviewerSelect
              pullRequestMetadata={pullRequestMetadata}
              onSelect={function (id: number): void {
                updateCodeCommentStatus({ reviewer_id: id }).catch(err => {
                  showError(getErrorMessage(err))
                })
                if (refetchReviewers) {
                  refetchReviewers()
                }
              }}></ReviewerSelect>
            {/* </Popover> */}
          </Layout.Horizontal>
          <Container padding={{ top: 'medium', bottom: 'large' }}>
            {/* <Text
              className={css.semiBoldText}
              padding={{ bottom: 'medium' }}
              font={{ variation: FontVariation.FORM_LABEL, size: 'small' }}>
              {getString('required')}
            </Text> */}
            {reviewers && reviewers?.length !== 0 ? (
              reviewers.map(
                (reviewer: {
                  reviewer: { display_name: string; id: number }
                  review_decision: EnumPullReqReviewDecision
                  sha: string
                }): Unknown => {
                  const updatedReviewDecision = processReviewDecision(
                    reviewer.review_decision,
                    reviewer.sha,
                    pullRequestMetadata?.source_sha
                  )
                  const reviewerInfo = generateReviewDecisionInfo(updatedReviewDecision)
                  return (
                    <Layout.Horizontal key={reviewer.reviewer.id} className={css.alignLayout}>
                      <Utils.WrapOptionalTooltip
                        tooltip={
                          <Text color={Color.GREY_100} padding="small">
                            {reviewerInfo.message}
                          </Text>
                        }
                        tooltipProps={{ isDark: true, interactionKind: PopoverInteractionKind.HOVER }}>
                        {updatedReviewDecision === PullReqReviewDecision.outdated ? (
                          <img className={css.svgOutdated} src={ignoreFailed} width={20} height={20} />
                        ) : (
                          <Icon {...omit(reviewerInfo, 'iconProps')} />
                        )}
                      </Utils.WrapOptionalTooltip>
                      <Avatar
                        className={cx(css.reviewerAvatar, {
                          [css.iconPadding]: updatedReviewDecision !== PullReqReviewDecision.changeReq
                        })}
                        name={reviewer.reviewer.display_name}
                        size="small"
                        hoverCard={false}
                      />

                      <Text lineClamp={1} className={css.reviewerName}>
                        {reviewer.reviewer.display_name}
                      </Text>
                      <FlexExpander />
                      <OptionsMenuButton
                        isDark={true}
                        icon="Options"
                        iconProps={{ size: 14 }}
                        style={{ paddingBottom: '9px' }}
                        // disabled={!!commentItem?.deleted}
                        width="100px"
                        height="24px"
                        items={[
                          // {
                          //   text: getString('makeOptional'),
                          //   onClick: noop
                          // },
                          // {
                          //   text: getString('makeRequired'),
                          //   onClick: noop
                          // },
                          // '-',
                          {
                            isDanger: true,
                            text: getString('remove'),
                            onClick: () => {
                              removeReviewer({}, { pathParams: { id: reviewer.reviewer.id } }).catch(err => {
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
                }
              )
            ) : (
              <Text color={Color.GREY_300} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                {getString('noReviewers')}
              </Text>
            )}
            {/* <Text
              className={css.semiBoldText}
              padding={{ top: 'medium', bottom: 'medium' }}
              font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
              {getString('optional')}
            </Text>
            {reviewers && reviewers?.length !== 0 ? (
              reviewers.map((reviewer: { reviewer: { display_name: string; id: number }; review_decision: string }) => {
                return (
                  <Layout.Horizontal key={reviewer.reviewer.id}>
                    <Icon className={css.reviewerPadding} name="dot" />
                    <Avatar
                      className={css.reviewerAvatar}
                      name={reviewer.reviewer.display_name}
                      size="small"
                      hoverCard={false}
                    />

                    <Text className={css.reviewerName}>{reviewer.reviewer.display_name}</Text>
                    <FlexExpander />
                    <OptionsMenuButton
                      isDark={true}
                      icon="Options"
                      iconProps={{ size: 14 }}
                      style={{ paddingBottom: '9px' }}
                      // disabled={!!commentItem?.deleted}
                      width="100px"
                      height="24px"
                      items={[
                        {
                          text: getString('makeOptional'),
                          onClick: noop
                        },
                        {
                          text: getString('makeRequired'),
                          onClick: noop
                        },
                        '-',
                        {
                          isDanger: true,
                          text: getString('remove'),
                          onClick: noop
                        }
                      ]}
                    />
                  </Layout.Horizontal>
                )
              })
            ) : (
              <Text color={Color.GREY_300} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                {getString('noOptionalReviewers')}
              </Text>
            )} */}
          </Container>
          {/* <Layout.Horizontal>
            <Text style={{ lineHeight: '24px' }} font={{ variation: FontVariation.H6 }}>
              {getString('tags')}
            </Text>
            <FlexExpander />
            <Button text={'Add +'} size={ButtonSize.SMALL} variation={ButtonVariation.TERTIARY}></Button>
          </Layout.Horizontal>
          {tagArr.length !== 0 ? (
            <></>
          ) : (
            <Text
              font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}
              padding={{ top: 'large', bottom: 'large' }}>
              {getString('noneYet')}
            </Text>
          )} */}
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

export default PullRequestSideBar
