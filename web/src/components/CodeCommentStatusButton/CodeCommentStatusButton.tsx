import React, { useMemo, useState } from 'react'
import { useMutate } from 'restful-react'
import { useToaster, Button, ButtonVariation, ButtonSize, useIsMounted } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReqActivity } from 'services/code'
import { useEmitCodeCommentStatus } from 'hooks/useEmitCodeCommentStatus'
import { CodeCommentState, getErrorMessage } from 'utils/Utils'
import type { CommentItem } from '../CommentBox/CommentBox'

interface CodeCommentStatusButtonProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  commentItems: CommentItem<TypesPullReqActivity>[]
  onCommentUpdate: () => void
}

export const CodeCommentStatusButton: React.FC<CodeCommentStatusButtonProps> = ({
  repoMetadata,
  pullRequestMetadata,
  commentItems,
  onCommentUpdate
}) => {
  const isMounted = useIsMounted()
  const { getString } = useStrings()
  const { showError } = useToaster()
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/comments`,
    [repoMetadata.path, pullRequestMetadata?.number]
  )
  const { mutate: updateCodeCommentStatus } = useMutate({ verb: 'PUT', path: ({ id }) => `${path}/${id}/status` })
  const [resolved, setResolved] = useState(commentItems[0]?.payload?.resolved ? true : false)
  const emitCodeCommentStatus = useEmitCodeCommentStatus({
    id: commentItems[0]?.payload?.id,
    onMatch: status => {
      if (isMounted.current) {
        const isResolved = status === CodeCommentState.RESOLVED
        setResolved(isResolved)

        if (commentItems[0]?.payload) {
          if (isResolved) {
            commentItems[0].payload.resolved = Date.now()
          } else {
            commentItems[0].payload.resolved = 0
          }
        }
      }
    }
  })

  return (
    <Button
      text={getString(resolved ? 'reactivate' : 'resolve')}
      variation={ButtonVariation.TERTIARY}
      size={ButtonSize.MEDIUM}
      onClick={async () => {
        const status = resolved ? CodeCommentState.ACTIVE : CodeCommentState.RESOLVED
        const payload = { status }
        const id = commentItems[0]?.payload?.id

        updateCodeCommentStatus(payload, { pathParams: { id } })
          .then(() => {
            onCommentUpdate()
            emitCodeCommentStatus(status)

            if (isMounted.current) {
              setResolved(!resolved)
            }
          })
          .catch(_exception => {
            showError(getErrorMessage(_exception), 0, getString('pr.failedToUpdateCommentStatus'))
          })
      }}></Button>
  )
}
