import {
  ButtonVariation,
  Color,
  Container,
  Icon,
  IconName,
  SplitButton,
  useToaster,
  Text,
  FontVariation,
  Layout
} from '@harness/uicore'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import cx from 'classnames'
import { useMutate } from 'restful-react'
import React, { useCallback, useState } from 'react'
import { useStrings } from 'framework/strings'
import type { EnumPullReqReviewDecision, TypesPullReq } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { getErrorMessage } from 'utils/Utils'
import css from '../Changes.module.scss'

interface PrReviewOption {
  method: EnumPullReqReviewDecision | 'reject'
  title: string
  disabled?: boolean
  icon: IconName
  color: Color
}

interface ReviewSplitButtonProps extends Pick<GitInfoProps, 'repoMetadata'> {
  shouldHide: boolean
  pullRequestMetadata?: TypesPullReq
}
const ReviewSplitButton = (props: ReviewSplitButtonProps) => {
  const { pullRequestMetadata, repoMetadata, shouldHide } = props
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()

  const prDecisionOptions: PrReviewOption[] = [
    {
      method: 'approved',
      title: getString('approve'),
      icon: 'tick-circle' as IconName,
      color: Color.GREEN_700
    },
    {
      method: 'changereq',
      title: getString('requestChanges'),
      icon: 'error' as IconName,
      color: Color.ORANGE_700
    },
    {
      method: 'reject',
      title: getString('reject'),
      disabled: true,
      icon: 'danger-icon' as IconName,
      color: Color.RED_700
    }
  ]

  const [decisionOption, setDecisionOption] = useState<PrReviewOption>(prDecisionOptions[0])
  const { mutate, loading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviews`
  })
  const submitReview = useCallback(() => {
    mutate({ decision: decisionOption.method })
      .then(() => {
        // setReset(true)
        showSuccess(getString('pr.reviewSubmitted'))
      })
      .catch(exception => showError(getErrorMessage(exception)))
  }, [decisionOption, mutate, showError, showSuccess, getString])
  return (
    <Container className={cx(css.btn, { [css.hide]: shouldHide })}>
      <SplitButton
        text={decisionOption.title}
        disabled={loading}
        variation={ButtonVariation.SECONDARY}
        popoverProps={{
          interactionKind: 'click',
          usePortal: true,
          popoverClassName: css.popover,
          position: PopoverPosition.BOTTOM_RIGHT,
          transitionDuration: 1000
        }}
        onClick={() => {
          submitReview()
        }}>
        {prDecisionOptions.map(option => {
          return (
            <Menu.Item
              key={option.method}
              className={css.menuReviewItem}
              disabled={option.disabled}
              text={
                <Layout.Horizontal>
                  <Icon
                    className={css.reviewIcon}
                    {...(option.icon === 'danger-icon' ? null : { color: option.color })}
                    size={16}
                    name={option.icon}
                  />
                  <Text flex width={'fit-content'} font={{ variation: FontVariation.BODY }}>
                    {option.title}
                  </Text>
                </Layout.Horizontal>
              }
              onClick={() => {
                setDecisionOption(option)
              }}
            />
          )
        })}
      </SplitButton>
    </Container>
  )
}

export default ReviewSplitButton
