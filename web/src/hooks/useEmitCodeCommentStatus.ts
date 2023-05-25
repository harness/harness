import { useCallback, useEffect } from 'react'
import type { CodeCommentState } from 'utils/Utils'

const PR_COMMENT_STATUS_CHANGED_EVENT = 'PR_COMMENT_STATUS_CHANGED_EVENT'
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
