import React, { useMemo, useState } from 'react'
import { useMutate } from 'restful-react'
import { useToaster, Button, ButtonVariation, ButtonSize, ButtonProps, useIsMounted } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReqActivity } from 'services/code'
import { useEmitCodeCommentStatus } from 'hooks/useEmitCodeCommentStatus'
import { CodeCommentState, getErrorMessage } from 'utils/Utils'
import type { CommentItem } from '../CommentBox/CommentBox'

interface CodeCommentSecondarySaveButtonProps
  extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>,
    ButtonProps {
  commentItems: CommentItem<TypesPullReqActivity>[]
}

export const CodeCommentSecondarySaveButton: React.FC<CodeCommentSecondarySaveButtonProps> = ({
  repoMetadata,
  pullRequestMetadata,
  commentItems,
  onClick,
  ...props
}) => {
  const { getString } = useStrings()
  const isMounted = useIsMounted()
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
        setResolved(status === CodeCommentState.RESOLVED)
      }
    }
  })

  return (
    <Button
      text={getString(resolved ? 'replyAndReactivate' : 'replyAndResolve')}
      variation={ButtonVariation.TERTIARY}
      size={ButtonSize.MEDIUM}
      onClick={async () => {
        const status = resolved ? CodeCommentState.ACTIVE : CodeCommentState.RESOLVED
        const payload = { status }
        const id = commentItems[0]?.payload?.id

        updateCodeCommentStatus(payload, { pathParams: { id } })
          .then(() => {
            if (commentItems[0]?.payload) {
              if (resolved) {
                commentItems[0].payload.resolved = 0
              } else {
                commentItems[0].payload.resolved = Date.now()
              }
            }
            if (isMounted.current) {
              setResolved(!resolved)
            }
            emitCodeCommentStatus(status)
            ;(onClick as () => void)()
          })
          .catch(_exception => {
            showError(getErrorMessage(_exception), 0, getString('pr.failedToUpdateCommentStatus'))
          })
      }}
      {...props}
    />
  )
}
