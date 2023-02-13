import React from 'react'
import {
  Container,
  Layout,
  Text,
  Button,
  FontVariation,
  ButtonVariation,
  Icon,
  Avatar,
  FlexExpander,
  ButtonSize,
  Color,
  IconName
} from '@harness/uicore'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useStrings } from 'framework/strings'
import css from './PullRequestSideBar.module.scss'

interface PullRequestSideBarProps {
  reviewers?: any
}

const PullRequestSideBar = (props: PullRequestSideBarProps) => {
  const { reviewers } = props
  const { getString } = useStrings()
  const tagArr = []

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
  //TODO add actions when you click the options menu button and also api integration
  return (
    <Container width={`30%`} padding={{ left: 'xxlarge', right: 'xxlarge' }}>
      <Container padding={{ left: 'xxlarge' }}>
        <Layout.Vertical>
          <Layout.Horizontal>
            <Text style={{ lineHeight: '24px' }} font={{ variation: FontVariation.H6 }}>
              {getString('reviewers')}
            </Text>
            <FlexExpander />
            <Button variation={ButtonVariation.TERTIARY} size={ButtonSize.SMALL} text={'Add +'}></Button>
          </Layout.Horizontal>
          <Container padding={{ top: 'medium', bottom: 'large' }}>
            <Text padding={{ bottom: 'medium' }} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
              {getString('required')}
            </Text>
            {reviewers && reviewers?.length !== 0 ? (
              reviewers.map(
                (reviewer: { reviewer: { display_name: string; id: number }; review_decision: string }): any => {
                  return (
                    <Layout.Horizontal key={reviewer.reviewer.id}>
                      <Icon className={css.reviewerPadding} {...generateReviewDecisionIcon(reviewer.review_decision)} />
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
                            onClick: () => {}
                          },
                          {
                            text: getString('makeRequired'),
                            onClick: () => {}
                          },
                          '-',
                          {
                            isDanger: true,
                            text: getString('remove'),
                            onClick: () => {}
                          }
                        ]}
                      />
                    </Layout.Horizontal>
                  )
                }
              )
            ) : (
              <Text
                className={css.noReviewerText}
                color={Color.GREY_300}
                font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                {getString('noRequiredReviewers')}
              </Text>
            )}
            <Text
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
                          onClick: () => {}
                        },
                        {
                          text: getString('makeRequired'),
                          onClick: () => {}
                        },
                        '-',
                        {
                          isDanger: true,
                          text: getString('remove'),
                          onClick: () => {}
                        }
                      ]}
                    />
                  </Layout.Horizontal>
                )
              })
            ) : (
              <Text
                className={css.noReviewerText}
                color={Color.GREY_300}
                font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                {getString('noOptionalReviewers')}
              </Text>
            )}
          </Container>
          <Layout.Horizontal>
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
          )}
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

export default PullRequestSideBar
