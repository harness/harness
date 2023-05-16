import type { TypesPullReqActivity } from 'services/code'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import { CommentType } from 'components/DiffViewer/DiffViewerUtils'

export function isCodeComment(commentItems: CommentItem<TypesPullReqActivity>[]) {
  return commentItems[0]?.payload?.type === CommentType.CODE_COMMENT
}

export function isSystemComment(commentItems: CommentItem<TypesPullReqActivity>[]) {
  return commentItems[0].payload?.kind === 'system'
}
