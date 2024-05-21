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

import React, { useEffect, useMemo, useState } from 'react'
import { useMutate } from 'restful-react'
import { random } from 'lodash-es'
import cx from 'classnames'
import { useToaster, Select } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReqActivity } from 'services/code'
import { CodeCommentState, getErrorMessage } from 'utils/Utils'
import { useEmitCodeCommentStatus } from 'hooks/useEmitCodeCommentStatus'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import css from './CodeCommentStatusSelect.module.scss'

interface CodeCommentStatusSelectProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  comment: { commentItems: CommentItem<TypesPullReqActivity>[] }
  rowElement?: HTMLTableRowElement
}

export const CodeCommentStatusSelect: React.FC<CodeCommentStatusSelectProps> = ({
  repoMetadata,
  pullReqMetadata,
  comment: { commentItems },
  rowElement
}) => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata?.number}/comments`,
    [repoMetadata.path, pullReqMetadata?.number]
  )
  const { mutate: updateCodeCommentStatus } = useMutate({ verb: 'PUT', path: ({ id }) => `${path}/${id}/status` })
  const commentStatus = useMemo(
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
  const [parentComment, setParentComment] = useState(commentItems[0])
  const randomClass = useMemo(() => `CodeCommentStatusSelect-${random(1_000_000, false)}`, [])
  const [codeCommentStatus, setCodeCommentStatus] = useState(
    parentComment?.payload?.resolved ? commentStatus[1] : commentStatus[0]
  )
  const emitCodeCommentStatus = useEmitCodeCommentStatus({
    id: parentComment?.id,
    onMatch: status => {
      setCodeCommentStatus(status === CodeCommentState.ACTIVE ? commentStatus[0] : commentStatus[1])
    }
  })

  useEffect(() => {
    // Comment thread has been just created, check if parentComment is
    // not set up properly, then query the comment id from DOM and construct
    // it from scratch (this is a workaround for new comment thread).
    if (!parentComment?.id) {
      const id = document
        .querySelector(`.${randomClass}`)
        ?.closest('[data-comment-thread-id]')
        ?.getAttribute('data-comment-thread-id')

      if (id) {
        setParentComment({
          id: Number(id),
          payload: { resolved: 0 }
        } as CommentItem<TypesPullReqActivity>)
      }
    }
  }, [parentComment?.id, randomClass])

  useEffect(
    function updateRowElement() {
      if (rowElement) {
        rowElement.dataset.commentThreadStatus = codeCommentStatus.value
      }
    },
    [rowElement, codeCommentStatus]
  )

  return parentComment?.deleted ? null : (
    <Select
      className={cx(css.select, randomClass)}
      items={commentStatus}
      value={codeCommentStatus}
      onChange={newState => {
        const status = newState.value as CodeCommentState
        const payload = { status }
        const id = parentComment?.id
        const isActive = status === CodeCommentState.ACTIVE

        updateCodeCommentStatus(payload, { pathParams: { id } })
          .then(data => {
            if (!parentComment?.id) {
              setParentComment(data)
            }

            setCodeCommentStatus(isActive ? commentStatus[0] : commentStatus[1])
            emitCodeCommentStatus(status)

            if (parentComment?.payload) {
              if (isActive) {
                parentComment.payload.resolved = 0
              } else {
                parentComment.payload.resolved = Date.now()
              }
            }
          })
          .catch(_exception => {
            showError(getErrorMessage(_exception), 0, getString('pr.failedToUpdateCommentStatus'))
          })
      }}
    />
  )
}
