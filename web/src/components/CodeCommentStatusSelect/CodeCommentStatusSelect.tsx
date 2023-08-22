import React, { useMemo, useState } from 'react'
import { useMutate } from 'restful-react'
import { useToaster, Select } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReqActivity } from 'services/code'
import { CodeCommentState, getErrorMessage } from 'utils/Utils'
import { useEmitCodeCommentStatus } from 'hooks/useEmitCodeCommentStatus'
import type { CommentItem } from '../CommentBox/CommentBox'
import css from './CodeCommentStatusSelect.module.scss'

interface CodeCommentStatusSelectProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  commentItems: CommentItem<TypesPullReqActivity>[]
  onCommentUpdate: () => void

  refetchActivities?: () => void
}

export const CodeCommentStatusSelect: React.FC<CodeCommentStatusSelectProps> = ({
  repoMetadata,
  pullRequestMetadata,
  commentItems,
  onCommentUpdate,
  refetchActivities
}) => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/comments`,
    [repoMetadata.path, pullRequestMetadata?.number]
  )
  const { mutate: updateCodeCommentStatus } = useMutate({ verb: 'PUT', path: ({ id }) => `${path}/${id}/status` })
  const codeCommentStatusItems = useMemo(
    () => [
      {
        label: getString('active'),
        value: CodeCommentState.ACTIVE
      },
      {
        label: getString('resolved'),
        value: CodeCommentState.RESOLVED
      }
    ],
    [getString]
  )
  const [codeCommentStatus, setCodeCommentStatus] = useState(
    commentItems[0]?.payload?.resolved ? codeCommentStatusItems[1] : codeCommentStatusItems[0]
  )
  const emitCodeCommentStatus = useEmitCodeCommentStatus({
    id: commentItems[0]?.payload?.id,
    onMatch: status => {
      setCodeCommentStatus(status === CodeCommentState.ACTIVE ? codeCommentStatusItems[0] : codeCommentStatusItems[1])
    }
  })

  return (
    <Select
      className={css.select}
      items={codeCommentStatusItems}
      value={codeCommentStatus}
      onChange={newState => {
        const status = newState.value as CodeCommentState
        const payload = { status }
        const id = commentItems[0]?.payload?.id
        const isActive = status === CodeCommentState.ACTIVE

        updateCodeCommentStatus(payload, { pathParams: { id } })
          .then(() => {
            onCommentUpdate()
            setCodeCommentStatus(isActive ? codeCommentStatusItems[0] : codeCommentStatusItems[1])
            emitCodeCommentStatus(status)

            if (commentItems[0]?.payload) {
              if (isActive) {
                commentItems[0].payload.resolved = 0
              } else {
                commentItems[0].payload.resolved = Date.now()
              }
            }
            if (refetchActivities) {
              refetchActivities()
            }
          })
          .catch(_exception => {
            showError(getErrorMessage(_exception), 0, getString('pr.failedToUpdateCommentStatus'))
          })
      }}
    />
  )
}
