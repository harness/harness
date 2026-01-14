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

import React, { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { ButtonSize, Container, Layout, Text } from '@harnessio/uicore'
import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import * as Diff2Html from 'diff2html'
import { get } from 'lodash-es'
import type { TypesPullReqActivity } from 'services/code'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import { DIFF2HTML_CONFIG, ViewStyle } from 'components/DiffViewer/DiffViewerUtils'
import { useAppContext } from 'AppContext'
import { CodeIcon, type GitInfoProps } from 'utils/GitUtils'
import { PullRequestSection } from 'utils/Utils'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { isCodeComment } from '../PullRequestUtils'
import css from './Conversation.module.scss'

interface CodeCommentHeaderProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  commentItems: CommentItem<TypesPullReqActivity>[]
  threadId: number | undefined
}

export const CodeCommentHeader: React.FC<CodeCommentHeaderProps> = ({
  commentItems,
  threadId,
  repoMetadata,
  pullReqMetadata
}) => {
  const { routes } = useAppContext()
  const _isCodeComment = isCodeComment(commentItems) && !commentItems[0].deleted
  const id = `code-comment-snapshot-${threadId}`

  useEffect(() => {
    if (_isCodeComment) {
      // Note: Since payload does not have information about the file path, mode, and index, and we
      // don't render them anyway in the UI, we just use dummy info for them.
      const codeDiffSnapshot = [
        `diff --git a/src b/dest`,
        `new file mode 100644`,
        'index 0000000..0000000',
        `--- a/src/${get(commentItems[0], 'payload.code_comment.path')}`,
        `+++ b/dest/${get(commentItems[0], 'payload.code_comment.path')}`,
        get(commentItems[0], 'payload.payload.title', ''),
        ...get(commentItems[0], 'payload.payload.lines', [])
      ].join('\n')

      new Diff2HtmlUI(
        document.getElementById(id) as HTMLElement,
        Diff2Html.parse(codeDiffSnapshot, DIFF2HTML_CONFIG),
        Object.assign({}, DIFF2HTML_CONFIG, { outputFormat: ViewStyle.LINE_BY_LINE })
      ).draw()
    }
  }, [id, commentItems, _isCodeComment, threadId])

  return _isCodeComment ? (
    <Container className={css.snapshot}>
      <Layout.Vertical>
        <Container className={css.title}>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Text
              inline
              className={css.fname}
              lineClamp={1}
              tooltipProps={{
                portalClassName: css.popover
              }}>
              <Link
                // className={css.fname}
                to={`${routes.toCODEPullRequest({
                  repoPath: repoMetadata?.path as string,
                  pullRequestId: String(pullReqMetadata?.number),
                  pullRequestSection: PullRequestSection.FILES_CHANGED
                })}?path=${commentItems[0].payload?.code_comment?.path}&commentId=${commentItems[0].payload?.id}`}>
                {commentItems[0].payload?.code_comment?.path}
              </Link>
            </Text>
            {commentItems[0].payload?.code_comment?.path && (
              <CopyButton
                content={commentItems[0].payload?.code_comment?.path}
                icon={CodeIcon.Copy}
                size={ButtonSize.MEDIUM}
              />
            )}
          </Layout.Horizontal>
        </Container>
        <Container className={css.snapshotContent} id={id} />
      </Layout.Vertical>
    </Container>
  ) : null
}
