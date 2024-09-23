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

import type * as Diff2Html from 'diff2html'
import HoganJsUtils from 'diff2html/lib/hoganjs-utils'
import { get } from 'lodash-es'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import type { TypesPullReqActivity } from 'services/code'
import { FILE_VIEWED_OBSOLETE_SHA } from 'utils/GitUtils'

export enum ViewStyle {
  SIDE_BY_SIDE = 'side-by-side',
  LINE_BY_LINE = 'line-by-line'
}

export enum CommentType {
  COMMENT = 'comment',
  CODE_COMMENT = 'code-comment',
  TITLE_CHANGE = 'title-change',
  REVIEW_SUBMIT = 'review-submit',
  MERGE = 'merge',
  BRANCH_UPDATE = 'branch-update',
  BRANCH_DELETE = 'branch-delete',
  BRANCH_RESTORE = 'branch-restore',
  STATE_CHANGE = 'state-change',
  LABEL_MODIFY = 'label-modify',
  REVIEWER_ADD = 'reviewer-add',
  REVIEWER_DELETE = 'reviewer-delete'
}

export enum LabelActivity {
  ASSIGN = 'assign',
  UN_ASSIGN = 'unassign',
  RE_ASSIGN = 'reassign'
}

export enum ReviewerAddActivity {
  REQUESTED = 'requested',
  ASSIGNED = 'assigned',
  SELF_ASSIGNED = 'self_assigned'
}

/**
 * @deprecated
 */
export interface PullRequestCodeCommentPayload {
  type: CommentType
  version: string // used to avoid rendering old payload structure
  file_id: string // unique id of the changed file
  file_title: string
  language: string
  is_on_left: boolean // comment made on the left side pane
  at_line_number: number
  line_number_range: number[]
  range_text_content: string // raw text content where the comment is made
  diff_html_snapshot: string // snapshot used to render diff in comment (PR Conversation). Could be used to send email notification too (with more work on capturing CSS styles and put them inline)
}

export const DIFF_VIEWER_HEADER_HEIGHT = 48
// const DIFF_MAX_CHANGES = 100
// const DIFF_MAX_LINE_LENGTH = 100

export interface DiffCommentItem<T = Unknown> {
  inner: T
  left: boolean
  right: boolean
  lineNumberStart: number
  lineNumberEnd: number
  span: number
  commentItems: CommentItem<T>[]
  _commentItems?: CommentItem<T>[]
  filePath: string
  codeBlockContent?: string
  destroy: (() => void) | undefined
}

export const DIFF2HTML_CONFIG = {
  outputFormat: 'side-by-side',
  drawFileList: false,
  fileListStartVisible: false,
  fileContentToggle: true,
  matching: 'lines',
  synchronisedScroll: true,
  highlight: true,
  renderNothingWhenEmpty: false,
  compiledTemplates: {
    'generic-line': HoganJsUtils.compile(`
      <tr>
        <td class="{{lineClass}} {{type}}">
          <div style="position: relative; z-index: 100; width: 0px; height: 0px; display: inline-block;">
            <span class="annotation-for-line" data-annotation-for-line="{{lineNumber}}" tab-index="0" role="button">+</span>
            <span data-selected-indicator></span>
          </div>{{{lineNumber}}}<!-- {{{filePath}}} --></td>
        <td class="{{type}}" data-content-for-line-number="{{lineNumber}}" data-content-for-file-path="{{file.filePath}}">
            <div class="{{contentClass}}" style="position: relative; z-index: 1;">
            {{#prefix}}
              <span class="d2h-code-line-prefix">{{{prefix}}}</span>
            {{/prefix}}
            {{^prefix}}
              <span class="d2h-code-line-prefix">&nbsp;</span>
            {{/prefix}}
            {{#content}}
              <span class="d2h-code-line-ctn">{{{content}}}</span>
            {{/content}}
            {{^content}}
              <span class="d2h-code-line-ctn"><br></span>
            {{/content}}
            </div>
        </td>
      </tr>
    `),
    'side-by-side-file-diff': HoganJsUtils.compile(`
      <div id="{{fileHtmlId}}" data="{{file.filePath}}" class="d2h-file-wrapper side-by-side-file-diff" data-lang="{{file.language}}">
        <div class="d2h-file-header">
          {{{filePath}}}
        </div>
        <div class="d2h-files-diff">
            <div class="d2h-file-side-diff left">
                <div class="d2h-code-wrapper">
                    <table class="d2h-diff-table" cellpadding="0px" cellspacing="0px">
                        <tbody class="d2h-diff-tbody">
                        {{{diffs.left}}}
                        </tbody>
                    </table>
                </div>
            </div>
            <div class="d2h-file-side-diff right">
                <div class="d2h-code-wrapper">
                    <table class="d2h-diff-table" cellpadding="0px" cellspacing="0px">
                        <tbody class="d2h-diff-tbody">
                        {{{diffs.right}}}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
      </div>
    `),
    'line-by-line-file-diff': HoganJsUtils.compile(`
      <div id="{{fileHtmlId}}"  data="{{file.filePath}}" class="d2h-file-wrapper {{file.filePath}} line-by-line-file-diff" data-lang="{{file.language}}">
        <div class="d2h-file-header">
        {{{filePath}}}
        </div>
        <div class="d2h-file-diff">
            <div class="d2h-code-wrapper">
                <table class="d2h-diff-table" cellpadding="0px" cellspacing="0px">
                    <tbody class="d2h-diff-tbody">
                    {{{diffs}}}
                    </tbody>
                </table>
            </div>
        </div>
      </div>
    `),
    'line-by-line-numbers': HoganJsUtils.compile(`
      <div class="line-num1" data-line-number="{{oldNumber}}">{{oldNumber}}</div>
      <div class="line-num2" data-line-number="{{newNumber}}">{{newNumber}}</div>
    `)
  }
} as Readonly<Diff2Html.Diff2HtmlConfig>

export function getCommentLineInfo(
  comment: DiffCommentItem<TypesPullReqActivity>,
  contentDOM: HTMLDivElement | null,
  commentEntry: DiffCommentItem,
  viewStyle: ViewStyle
) {
  const isSideBySideView = viewStyle === ViewStyle.SIDE_BY_SIDE
  const { left, lineNumberEnd, filePath } = commentEntry
  const filePathBody = filePath ? contentDOM?.querySelector(`[data="${filePath}"`) : contentDOM

  const diffBody = filePathBody?.querySelector(
    `${isSideBySideView ? `.d2h-file-side-diff${left ? '.left' : '.right'} ` : ''}.d2h-diff-tbody`
  )
  const rowElement = (
    diffBody?.querySelector(`[data-content-for-line-number="${lineNumberEnd}"]`) ||
    diffBody?.querySelector(
      `${!isSideBySideView ? (left ? '.line-num1' : '.line-num2') : ''}[data-line-number="${lineNumberEnd}"]`
    )
  )?.closest('tr') as HTMLTableRowElement

  let node = rowElement as Element
  let rowPosition = 1
  while (node?.previousElementSibling) {
    rowPosition++
    node = node.previousElementSibling
  }

  let oppositeRowElement: HTMLTableRowElement | null = null

  if (isSideBySideView) {
    const filesDiff = rowElement?.closest('.d2h-files-diff') as HTMLElement
    const sideDiff = filesDiff?.querySelector(`div.${left ? 'right' : 'left'}`) as HTMLElement
    oppositeRowElement = sideDiff?.querySelector(`tr:nth-child(${rowPosition})`)
  }

  return {
    rowElement,
    rowPosition,
    commentThreadRendered: (rowElement?.dataset?.commentIds || '').split('/').includes(String(comment.inner.id)),
    oppositeRowElement
  }
}

export function createCommentOppositePlaceHolder(lineNumber: number, isNewCommentThread = false): HTMLTableRowElement {
  const placeHolderRow = document.createElement('tr')
  const initialHeight = `${isNewCommentThread ? NEW_COMMENT_BOX_INITIAL_HEIGHT : 0}px`

  placeHolderRow.dataset.placeHolderForLine = String(lineNumber)
  placeHolderRow.innerHTML = `
    <td height="${initialHeight}" class="d2h-code-side-linenumber d2h-code-side-emptyplaceholder d2h-cntx d2h-emptyplaceholder"></td>
    <td class="d2h-cntx d2h-emptyplaceholder" height="${initialHeight}">
      <div class="d2h-code-side-line d2h-code-side-emptyplaceholder">
        <span class="d2h-code-line-prefix">&nbsp;</span>
        <span class="d2h-code-line-ctn hljs"><br></span>
      </div>
    </td>
  `

  return placeHolderRow
}

export const activityToCommentItem = (activity: TypesPullReqActivity): CommentItem<TypesPullReqActivity> => ({
  id: activity.id || 0,
  author: activity.author?.display_name as string,
  created: activity.created as number,
  edited: activity.edited as number,
  updated: activity.updated as number,
  deleted: activity.deleted as number,
  outdated: !!activity.code_comment?.outdated,
  content: (activity.text || (activity.payload as Unknown)?.Message) as string,
  payload: activity
})

export function activitiesToDiffCommentItems(
  filePath: string,
  activities: TypesPullReqActivity[]
): DiffCommentItem<TypesPullReqActivity>[] {
  return (
    activities
      ?.filter(activity => filePath === activity.code_comment?.path)
      .map(activity => {
        const replyComments =
          activities
            ?.filter(replyActivity => replyActivity.parent_id === activity.id)
            .map(_activity => activityToCommentItem(_activity)) || []
        const right = get(activity.payload, 'line_start_new', false)
        const span = right ? activity.code_comment?.span_new || 0 : activity.code_comment?.span_old || 0
        const lineNumberStart = (right ? activity.code_comment?.line_new : activity.code_comment?.line_old) as number
        const lineNumberEnd = lineNumberStart + span - 1

        const diffSnapshotLines = get(activity.payload, 'lines', []) as string[]
        const leftLines: string[] = []
        const rightLines: string[] = []

        diffSnapshotLines.forEach(line => {
          const lineContent = line.substring(1) // line has a `prefix` (space, +, or -), always remove it

          if (line.startsWith('-')) {
            leftLines.push(lineContent)
          } else if (line.startsWith('+')) {
            rightLines.push(lineContent)
          } else {
            leftLines.push(lineContent)
            rightLines.push(lineContent)
          }
        })
        const diffHeader = get(activity.payload, 'title', '') as string
        const [oldStartLine, newStartLine] = diffHeader
          .replaceAll(/@|\+|-/g, '')
          .trim()
          .split(' ')
          .map(token => token.split(',')[0])
          .map(Number)
        const _startLine = right ? newStartLine : oldStartLine
        const codeLines = right ? rightLines : leftLines
        let lineIndex = 0

        while (lineIndex + _startLine < lineNumberStart) {
          lineIndex++
        }
        const codeBlockContent = codeLines
          .slice(lineNumberStart - _startLine, lineNumberStart - _startLine + lineNumberEnd - lineNumberStart + 1)
          .join('\n')

        return {
          inner: activity,
          left: !right,
          right,
          height: 0,
          lineNumberStart,
          lineNumberEnd,
          span,
          commentItems: [activityToCommentItem(activity)].concat(replyComments),
          filePath: filePath,
          destroy: undefined,
          eventStream: undefined,
          codeBlockContent
        }
      }) || []
  )
}

export enum FileViewedState {
  NOT_VIEWED,
  VIEWED,
  CHANGED
}

export function getFileViewedState(
  filePath: string,
  fileSha: string | undefined,
  views: Map<string, string> | undefined
): FileViewedState {
  if (!views || !views.has(filePath)) {
    return FileViewedState.NOT_VIEWED
  }

  const viewedSHA = views.get(filePath)

  // this case is only expected in case of pure rename - but we'll also use it as fallback.
  if (fileSha === undefined || fileSha === '') {
    return viewedSHA === FILE_VIEWED_OBSOLETE_SHA ? FileViewedState.CHANGED : FileViewedState.VIEWED
  }

  return viewedSHA === fileSha ? FileViewedState.VIEWED : FileViewedState.CHANGED
}

const NEW_COMMENT_BOX_INITIAL_HEIGHT = 241
