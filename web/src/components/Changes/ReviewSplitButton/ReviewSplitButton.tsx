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

import { ButtonVariation, Container, SplitButton, useToaster, Text, Layout } from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import cx from 'classnames'
import { useMutate } from 'restful-react'
import React, { useCallback, useMemo } from 'react'
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
  refreshPr: () => void
  disabled?: boolean
  refetchReviewers?: () => void
}
const ReviewSplitButton = (props: ReviewSplitButtonProps) => {
  const { refetchReviewers, pullRequestMetadata, repoMetadata, shouldHide, refreshPr, disabled } = props
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const prDecisionOptions: PrReviewOption[] = useMemo(
    () => [
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
      }
    ],
    [getString]
  )

  const { mutate, loading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviews`
  })
  const submitReview = useCallback(
    (decision: PrReviewOption) => {
      mutate({ decision: decision.method, commit_sha: pullRequestMetadata?.source_sha })
        .then(() => {
          showSuccess(getString(decision.method === 'approved' ? 'pr.reviewSubmitted' : 'pr.requestSubmitted'))
          refreshPr?.()
          refetchReviewers?.()
        })
        .catch(exception => showError(getErrorMessage(exception)))
    },
    [mutate, showError, showSuccess, getString, refreshPr, pullRequestMetadata?.source_sha, refetchReviewers]
  )
  return (
    <Container
      className={cx(css.reviewButton, {
        [css.hide]: shouldHide,
        [css.disabled]: disabled
      })}>
      <SplitButton
        text={prDecisionOptions[0].title}
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
          submitReview(prDecisionOptions[0])
        }}>
        <Menu.Item
          key={prDecisionOptions[1].method}
          className={css.menuReviewItem}
          disabled={disabled || prDecisionOptions[1].disabled}
          text={
            <Layout.Horizontal>
              <Icon
                className={css.reviewIcon}
                {...(prDecisionOptions[1].icon === 'danger-icon' ? null : { color: prDecisionOptions[1].color })}
                size={16}
                name={prDecisionOptions[1].icon}
              />
              <Text flex width={'fit-content'} font={{ variation: FontVariation.BODY }}>
                {prDecisionOptions[1].title}
              </Text>
            </Layout.Horizontal>
          }
          onClick={() => {
            submitReview(prDecisionOptions[1])
          }}
        />
      </SplitButton>
    </Container>
  )
}

export default ReviewSplitButton
