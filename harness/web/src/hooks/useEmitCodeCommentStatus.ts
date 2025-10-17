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

import { useCallback, useEffect } from 'react'
import type { CodeCommentState } from 'utils/Utils'

export const PR_COMMENT_STATUS_CHANGED_EVENT = 'PR_COMMENT_STATUS_CHANGED_EVENT'
export const PULL_REQUEST_ALL_COMMENTS_ID = -99999

interface UseEmitCodeCommentStatusProps {
  id?: number
  onMatch: (status: CodeCommentState, id: number) => void
}

export function useEmitCodeCommentStatus({ id, onMatch }: UseEmitCodeCommentStatusProps) {
  const callback = useCallback(
    event => {
      if ((id && event.detail.id === id) || id === PULL_REQUEST_ALL_COMMENTS_ID) {
        onMatch(event.detail.status, event.detail.id)
      }
    },
    [id, onMatch]
  )
  const emitCodeCommentStatus = useCallback(
    (status: CodeCommentState) => {
      const event = new CustomEvent(PR_COMMENT_STATUS_CHANGED_EVENT, { detail: { id, status } })
      document.dispatchEvent(event)
    },
    [id]
  )
  useEffect(() => {
    document.addEventListener(PR_COMMENT_STATUS_CHANGED_EVENT, callback)

    return () => {
      document.removeEventListener(PR_COMMENT_STATUS_CHANGED_EVENT, callback)
    }
  }, [callback])

  return emitCodeCommentStatus
}
