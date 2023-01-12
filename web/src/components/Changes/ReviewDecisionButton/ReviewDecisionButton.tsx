import React, { useCallback, useState } from 'react'
import { Radio, RadioGroup, ButtonVariation, Button, Container, Layout, ButtonSize, useToaster } from '@harness/uicore'
import { useMutate } from 'restful-react'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReq } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import css from './ReviewDecisionButton.module.scss'

enum PullReqReviewDecision {
  PENDING = 'pending',
  REVIEWED = 'reviewed',
  APPROVED = 'approved',
  CHANGEREQ = 'changereq'
}

interface ReviewDecisionButtonProps extends Pick<GitInfoProps, 'repoMetadata'> {
  shouldHide: boolean
  pullRequestMetadata?: TypesPullReq
}

export const ReviewDecisionButton: React.FC<ReviewDecisionButtonProps> = ({
  repoMetadata,
  pullRequestMetadata,
  shouldHide
}) => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const [comment, setComment] = useState('')
  const [decision, setDecision] = useState<PullReqReviewDecision>(PullReqReviewDecision.PENDING)
  const { mutate, loading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/review`
  })
  const submitReview = useCallback(() => {
    mutate({ decision, message: comment })
      .then(() => {
        alert('ok')
      })
      .catch(exception => showError(getErrorMessage(exception)))
  }, [comment, decision, mutate, showError])

  return (
    <Button
      text={getString('pr.reviewChanges')}
      variation={ButtonVariation.PRIMARY}
      intent="success"
      rightIcon="chevron-down"
      className={cx(css.btn, { [css.hide]: shouldHide })}
      style={{ '--background-color': 'var(--green-800)' } as React.CSSProperties}
      tooltip={
        <Container padding="large" className={css.popup}>
          <Layout.Vertical spacing="medium">
            <Container className={css.markdown} padding="medium">
              <MarkdownEditorWithPreview
                value={comment}
                onChange={setComment}
                hideButtons
                i18n={{
                  placeHolder: getString('leaveAComment'),
                  tabEdit: getString('write'),
                  tabPreview: getString('preview'),
                  save: getString('save'),
                  cancel: getString('cancel')
                }}
                editorHeight="100px"
              />
            </Container>
            <Container padding={{ left: 'xxxlarge' }}>
              <RadioGroup>
                <Radio
                  name="decision"
                  defaultChecked={decision === PullReqReviewDecision.PENDING}
                  label={getString('comment')}
                  value={PullReqReviewDecision.PENDING}
                  onChange={() => setDecision(PullReqReviewDecision.PENDING)}
                />
                <Radio
                  name="decision"
                  defaultChecked={decision === PullReqReviewDecision.APPROVED}
                  label={getString('approve')}
                  value={PullReqReviewDecision.APPROVED}
                  onChange={() => setDecision(PullReqReviewDecision.APPROVED)}
                />
                <Radio
                  name="decision"
                  defaultChecked={decision === PullReqReviewDecision.CHANGEREQ}
                  label={getString('requestChanges')}
                  value={PullReqReviewDecision.CHANGEREQ}
                  onChange={() => setDecision(PullReqReviewDecision.CHANGEREQ)}
                />
              </RadioGroup>
            </Container>
            <Container>
              <Button
                variation={ButtonVariation.PRIMARY}
                text={getString('submitReview')}
                size={ButtonSize.SMALL}
                onClick={submitReview}
                disabled={!(comment || '').trim().length}
                loading={loading}
              />
            </Container>
          </Layout.Vertical>
        </Container>
      }
      tooltipProps={{ interactionKind: 'click', position: 'bottom-right', hasBackdrop: true }}
    />
  )
}
