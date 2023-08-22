import React from 'react'
import { useMutate } from 'restful-react'
import { omit } from 'lodash-es'
import { Container, Layout, Text, Avatar, FlexExpander, useToaster } from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useStrings } from 'framework/strings'
import type { TypesPullReq, TypesRepository } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { ReviewerSelect } from 'components/ReviewerSelect/ReviewerSelect'
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

  const generateReviewDecisionIcon = (
    reviewDecision: string
  ): {
    name: IconName
    color: string | undefined
    size: number | undefined
    icon: IconName
    iconProps?: { color?: Color }
  } => {
    let icon: IconName = 'dot'
    let color: Color | undefined = undefined
    let size: number | undefined = undefined

    switch (reviewDecision) {
      case 'changereq':
        icon = 'cross'
        color = Color.RED_700
        size = 16
        break
      case 'approved':
        icon = 'tick'
        size = 16
        color = Color.GREEN_700
        break
      default:
        color = Color.GREY_100
        size = 16
    }
    const name = icon
    return { name, color, size, icon, ...(color ? { iconProps: { color } } : undefined) }
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
              gitRef={''}
              text={getString('add')}
              labelPrefix={getString('add')}
              onSelect={function (id: number): void {
                updateCodeCommentStatus({ reviewer_id: id }).catch(err => {
                  showError(getErrorMessage(err))
                })
                if (refetchReviewers) {
                  refetchReviewers()
                }
              }}
              repoMetadata={{}}></ReviewerSelect>
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
                (reviewer: { reviewer: { display_name: string; id: number }; review_decision: string }): Unknown => {
                  return (
                    <Layout.Horizontal key={reviewer.reviewer.id}>
                      <Icon
                        className={css.reviewerPadding}
                        {...omit(generateReviewDecisionIcon(reviewer.review_decision), 'iconProps')}
                      />
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
