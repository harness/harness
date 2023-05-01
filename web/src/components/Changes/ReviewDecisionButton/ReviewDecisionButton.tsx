// import React, { useCallback, useEffect, useState } from 'react'
// import { Radio, RadioGroup, ButtonVariation, Button, Container, Layout, ButtonSize, useToaster } from '@harness/uicore'
// import { Render } from 'react-jsx-match'
// import { useMutate } from 'restful-react'
// import cx from 'classnames'
// import { useStrings } from 'framework/strings'
// import type { GitInfoProps } from 'utils/GitUtils'
// import type { TypesPullReq } from 'services/code'
// import { getErrorMessage } from 'utils/Utils'
// import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
// import css from './ReviewDecisionButton.module.scss'

// enum PullReqReviewDecision {
//   REVIEWED = 'reviewed',
//   APPROVED = 'approved',
//   CHANGEREQ = 'changereq'
// }

// interface ReviewDecisionButtonProps extends Pick<GitInfoProps, 'repoMetadata'> {
//   shouldHide: boolean
//   pullRequestMetadata?: TypesPullReq
// }

// /**
//  * @deprecated
//  */
// export const ReviewDecisionButton: React.FC<ReviewDecisionButtonProps> = ({
//   repoMetadata,
//   pullRequestMetadata,
//   shouldHide
// }) => {
//   const { getString } = useStrings()
//   const { showError, showSuccess } = useToaster()
//   const [comment, setComment] = useState('')
//   const [reset, setReset] = useState(false)
//   const [decision, setDecision] = useState<PullReqReviewDecision>(PullReqReviewDecision.REVIEWED)
//   const { mutate, loading } = useMutate({
//     verb: 'POST',
//     path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviews`
//   })
//   const submitReview = useCallback(() => {
//     mutate({ decision, message: comment })
//       .then(() => {
//         setReset(true)
//         showSuccess(getString('pr.reviewSubmitted'))
//       })
//       .catch(exception => showError(getErrorMessage(exception)))
//   }, [comment, decision, mutate, showError, showSuccess, getString])

//   useEffect(() => {
//     let timeoutId = 0
//     if (reset) {
//       timeoutId = window.setTimeout(() => setReset(false))
//     }
//     return () => window.clearTimeout(timeoutId)
//   }, [reset])

//   return (
//     <Button
//       text={getString('pr.reviewChanges')}
//       variation={ButtonVariation.PRIMARY}
//       intent="success"
//       rightIcon="chevron-down"
//       className={cx(css.btn, { [css.hide]: shouldHide })}
//       style={{ '--background-color': 'var(--green-800)' } as React.CSSProperties}
//       tooltip={
//         <Render when={!reset}>
//           <Container padding="large" className={css.popup}>
//             <Layout.Vertical spacing="medium">
//               <Container className={css.markdown} padding="medium">
//                 <MarkdownEditorWithPreview
//                   value={comment}
//                   onChange={setComment}
//                   hideButtons
//                   i18n={{
//                     placeHolder: getString('leaveAComment'),
//                     tabEdit: getString('write'),
//                     tabPreview: getString('preview'),
//                     save: getString('save'),
//                     cancel: getString('cancel')
//                   }}
//                   editorHeight="100px"
//                 />
//               </Container>
//               <Container padding={{ left: 'xxxlarge' }}>
//                 <RadioGroup>
//                   <Radio
//                     name="decision"
//                     defaultChecked={decision === PullReqReviewDecision.REVIEWED}
//                     label={getString('comment')}
//                     value={PullReqReviewDecision.REVIEWED}
//                     onChange={() => setDecision(PullReqReviewDecision.REVIEWED)}
//                   />
//                   <Radio
//                     name="decision"
//                     defaultChecked={decision === PullReqReviewDecision.APPROVED}
//                     label={getString('approve')}
//                     value={PullReqReviewDecision.APPROVED}
//                     onChange={() => setDecision(PullReqReviewDecision.APPROVED)}
//                   />
//                   <Radio
//                     name="decision"
//                     defaultChecked={decision === PullReqReviewDecision.CHANGEREQ}
//                     label={getString('requestChanges')}
//                     value={PullReqReviewDecision.CHANGEREQ}
//                     onChange={() => setDecision(PullReqReviewDecision.CHANGEREQ)}
//                   />
//                 </RadioGroup>
//               </Container>
//               <Container>
//                 <Button
//                   variation={ButtonVariation.PRIMARY}
//                   text={getString('submitReview')}
//                   size={ButtonSize.MEDIUM}
//                   onClick={submitReview}
//                   disabled={!(comment || '').trim().length && decision != PullReqReviewDecision.APPROVED}
//                   loading={loading}
//                 />
//               </Container>
//             </Layout.Vertical>
//           </Container>
//         </Render>
//       }
//       tooltipProps={{ interactionKind: 'click', position: 'bottom-right', hasBackdrop: true }}
//     />
//   )
// }
