import React, { useState } from 'react'
import { Radio, RadioGroup, ButtonVariation, Button, Container, Layout, Text, ButtonSize } from '@harness/uicore'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReq } from 'services/code'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import css from './ReviewDecisionButton.module.scss'

enum PullReqReviewDecision {
  PENDING = 'pending',
  REVIEWED = 'reviewed',
  APPROVED = 'approved',
  CHANGEREQ = 'changereq'
}

interface ReviewDecisionButtonProps extends Pick<GitInfoProps, 'repoMetadata'> {
  disable: boolean
  pullRequestMetadata?: TypesPullReq
}

export const ReviewDecisionButton: React.FC<ReviewDecisionButtonProps> = ({
  pullRequestMetadata,
  repoMetadata,
  disable
}) => {
  const { getString } = useStrings()
  const [content, setContent] = useState('')
  const [decision, setDecision] = useState<PullReqReviewDecision>(PullReqReviewDecision.PENDING)

  return (
    <Button
      text={getString('pr.reviewChanges')}
      variation={ButtonVariation.PRIMARY}
      intent="success"
      rightIcon="chevron-down"
      className={cx(css.btn, { [css.hide]: disable })}
      style={{ '--background-color': 'var(--green-800)' } as React.CSSProperties}
      tooltip={
        <Container padding="large" className={css.popup}>
          <Layout.Vertical spacing="medium">
            <Text>Finish your review</Text>
            <Container className={css.markdown} padding="medium">
              <MarkdownEditorWithPreview
                value={content}
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
              <Button variation={ButtonVariation.PRIMARY} text={getString('submitReview')} size={ButtonSize.SMALL} />
            </Container>
          </Layout.Vertical>
        </Container>
      }
      tooltipProps={{ interactionKind: 'click', position: 'bottom-right', hasBackdrop: true }}
    />
  )
}
