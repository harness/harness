import type * as Diff2Html from 'diff2html'
import HoganJsUtils from 'diff2html/lib/hoganjs-utils'
import { get } from 'lodash-es'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import type { TypesPullReqActivity } from 'services/code'
import type { DiffFileEntry } from 'utils/types'

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
  STATE_CHANGE = 'state-change'
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

export const DIFF_VIEWER_HEADER_HEIGHT = 36
// const DIFF_MAX_CHANGES = 100
// const DIFF_MAX_LINE_LENGTH = 100

export interface DiffCommentItem<T = Unknown> {
  left: boolean
  right: boolean
  lineNumber: number
  height: number
  commentItems: CommentItem<T>[]
}

export const DIFF2HTML_CONFIG = {
  outputFormat: 'side-by-side',
  drawFileList: false,
  fileListStartVisible: false,
  fileContentToggle: true,
  // diffMaxChanges: DIFF_MAX_CHANGES,
  // diffMaxLineLength: DIFF_MAX_LINE_LENGTH,
  // diffTooBigMessage: index => `${index} - is too big`,
  matching: 'lines',
  synchronisedScroll: true,
  highlight: true,
  renderNothingWhenEmpty: false,
  compiledTemplates: {
    'generic-line': HoganJsUtils.compile(`
      <tr>
        <td class="{{lineClass}} {{type}}">
          {{{lineNumber}}}
        </td>
        <td class="{{type}}" data-content-for-line-number="{{lineNumber}}">
            <div data-annotation-for-line="{{lineNumber}}" tab-index="0" role="button">+</div>
            <div class="{{contentClass}}">
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
      <div id="{{fileHtmlId}}" class="d2h-file-wrapper side-by-side-file-diff" data-lang="{{file.language}}">
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
      <div id="{{fileHtmlId}}" class="d2h-file-wrapper line-by-line-file-diff" data-lang="{{file.language}}">
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
  contentDOM: HTMLDivElement | null,
  commentEntry: DiffCommentItem,
  viewStyle: ViewStyle
) {
  const isSideBySideView = viewStyle === ViewStyle.SIDE_BY_SIDE
  const { left, lineNumber } = commentEntry
  const diffBody = contentDOM?.querySelector(
    `${isSideBySideView ? `.d2h-file-side-diff${left ? '.left' : '.right'} ` : ''}.d2h-diff-tbody`
  )
  const rowElement = (
    diffBody?.querySelector(`[data-content-for-line-number="${lineNumber}"]`) ||
    diffBody?.querySelector(
      `${!isSideBySideView ? (left ? '.line-num1' : '.line-num2') : ''}[data-line-number="${lineNumber}"]`
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
    hasCommentsRendered: !!rowElement?.dataset?.annotated,
    oppositeRowElement
  }
}

export function renderCommentOppositePlaceHolder(annotation: DiffCommentItem, oppositeRowElement: HTMLTableRowElement) {
  const placeHolderRow = document.createElement('tr')

  placeHolderRow.dataset.placeHolderForLine = String(annotation.lineNumber)
  placeHolderRow.innerHTML = `
    <td height="${annotation.height}px" class="d2h-code-side-linenumber d2h-code-side-emptyplaceholder d2h-cntx d2h-emptyplaceholder"></td>
    <td class="d2h-cntx d2h-emptyplaceholder" height="${annotation.height}px">
      <div class="d2h-code-side-line d2h-code-side-emptyplaceholder">
        <span class="d2h-code-line-prefix">&nbsp;</span>
        <span class="d2h-code-line-ctn hljs"><br></span>
      </div>
    </td>
  `
  oppositeRowElement.after(placeHolderRow)
}

export const activityToCommentItem = (activity: TypesPullReqActivity): CommentItem<TypesPullReqActivity> => ({
  author: activity.author?.display_name as string,
  created: activity.created as number,
  updated: activity.edited as number,
  deleted: activity.deleted as number,
  outdated: !!activity.code_comment?.outdated,
  content: (activity.text || (activity.payload as Unknown)?.Message) as string,
  payload: activity
})

export function activitiesToDiffCommentItems(diff: DiffFileEntry): DiffCommentItem<TypesPullReqActivity>[] {
  return (
    diff.fileActivities?.map(activity => {
      const replyComments =
        diff.activities
          ?.filter(replyActivity => replyActivity.parent_id === activity.id)
          .map(_activity => activityToCommentItem(_activity)) || []
      const right = get(activity.payload, 'line_start_new', false)

      return {
        left: !right,
        right,
        height: 0,
        lineNumber: (right ? activity.code_comment?.line_new : activity.code_comment?.line_old) as number,
        commentItems: [activityToCommentItem(activity)].concat(replyComments)
      }
    }) || []
  )
}
