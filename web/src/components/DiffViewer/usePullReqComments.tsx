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

import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { useMutate } from 'restful-react'
import Selecto from 'selecto'
import ReactDOM from 'react-dom'
import { useLocation } from 'react-router-dom'
import { useToaster, ButtonProps, Utils } from '@harnessio/uicore'
import { findLastIndex, get, isEqual, max, noop, random, uniq } from 'lodash-es'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { DiffFileEntry } from 'utils/types'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useAppContext } from 'AppContext'
import type { OpenapiCommentCreatePullReqRequest, TypesPullReq, TypesPullReqActivity } from 'services/code'
import { CodeCommentState, PullRequestSection, filenameToLanguage, getErrorMessage } from 'utils/Utils'
import { AppWrapper } from 'App'
import { CodeCommentStatusButton } from 'components/CodeCommentStatusButton/CodeCommentStatusButton'
import { CodeCommentSecondarySaveButton } from 'components/CodeCommentSecondarySaveButton/CodeCommentSecondarySaveButton'
import { CodeCommentStatusSelect } from 'components/CodeCommentStatusSelect/CodeCommentStatusSelect'
import { dispatchCustomEvent } from 'hooks/useEventListener'
import { UseGetPullRequestInfoResult, usePullReqActivities } from 'pages/PullRequest/useGetPullRequestInfo'
import { CommentThreadTopDecoration } from 'components/CommentThreadTopDecoration/CommentThreadTopDecoration'
import {
  activitiesToDiffCommentItems,
  activityToCommentItem,
  CommentType,
  DiffCommentItem,
  getCommentLineInfo,
  createCommentOppositePlaceHolder,
  ViewStyle
} from './DiffViewerUtils'
import {
  CommentAction,
  CommentBox,
  CommentBoxOutletPosition,
  CommentItem,
  customEventForCommentWithId
} from '../CommentBox/CommentBox'
import css from './DiffViewer.module.scss'

interface UsePullReqCommentsProps extends Pick<GitInfoProps, 'repoMetadata'> {
  diff: DiffFileEntry
  viewStyle: ViewStyle
  stickyTopPosition?: number
  readOnly?: boolean
  pullReqMetadata?: TypesPullReq
  targetRef?: string
  sourceRef?: string
  commitRange?: string[]
  scrollElement: HTMLElement
  collapsed: boolean
  containerRef: React.RefObject<HTMLDivElement | null>
  contentRef: React.RefObject<HTMLDivElement | null>
  refetchActivities?: UseGetPullRequestInfoResult['refetchActivities']
  setDirty?: React.Dispatch<React.SetStateAction<boolean>>
}

export function usePullReqComments({
  diff,
  viewStyle,
  stickyTopPosition = 0,
  readOnly,
  repoMetadata,
  pullReqMetadata,
  targetRef,
  sourceRef,
  commitRange,
  scrollElement,
  collapsed,
  containerRef,
  contentRef,
  refetchActivities,
  setDirty
}: UsePullReqCommentsProps) {
  const activities = usePullReqActivities()
  const { getString } = useStrings()
  const { routingId, currentUser, standalone, routes } = useAppContext()
  const { showError } = useToaster()
  const confirmAct = useConfirmAct()
  const commentPath = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata?.number}/comments`,
    [repoMetadata.path, pullReqMetadata?.number]
  )
  const { save, update, remove } = useCommentAPI(commentPath)
  const location = useLocation()
  const [comments] = useState(new Map<number, DiffCommentItem<TypesPullReqActivity>>())
  const copyLinkToComment = useCallback(
    (id, commentItem) => {
      const path = `${routes.toCODEPullRequest({
        repoPath: repoMetadata?.path as string,
        pullRequestId: String(pullReqMetadata?.number),
        pullRequestSection: PullRequestSection.FILES_CHANGED
      })}?path=${commentItem.payload?.code_comment?.path}&commentId=${id}`
      const { pathname, origin } = window.location

      Utils.copy(origin + pathname.replace(location.pathname, '') + path)
    },
    [location, pullReqMetadata?.number, repoMetadata?.path, routes]
  )

  /**
   * Update data-selected-count attribute on a code row/line to keep track of how many
   * comments mark the line as being selected.
   */
  const updateDataSelectedCount = useCallback((row: HTMLElement | null, isSelected: boolean) => {
    if (row) {
      const selectedCount = Number(row.dataset.selectedCount || 0) + (isSelected ? 1 : -1)

      row.dataset.selectedCount = String(selectedCount)

      if (isSelected) {
        // Skip marking info lines (and others if needed)
        if (!row.querySelector('td.d2h-info')) {
          row.classList.add(selected)
        }
      } else {
        // When this number is zero, it's time to clear the selected state
        if (!(selectedCount > 0)) {
          row.classList.remove(selected)
          delete row.dataset.selectedCount
          row.classList.remove(selected)
        }
      }
    }
  }, [])

  const updateDataCommentIds = useCallback((row: HTMLElement, commentId: number, add = true) => {
    if (row && commentId) {
      let ids = (row.dataset.commentIds || '').split('/').filter(Boolean)
      const id = String(commentId)

      if (add && !ids.includes(id)) {
        ids.push(id)
      } else if (!add && ids.includes(id)) {
        ids = ids.filter(_id => _id !== id)
      }

      row.dataset.commentIds = ids.join('/')
    }
  }, [])

  /**
   * Mark lines associated with a comment thread `selected`, or not.
   */
  const markSelectedLines = useCallback(
    (comment: DiffCommentItem<TypesPullReqActivity>, atRow: HTMLTableRowElement | null, isSelected: boolean) => {
      const handledRows = new WeakMap() // use a map to make sure markers are not marked twice per comment
      const sideBySide = viewStyle === ViewStyle.SIDE_BY_SIDE

      if (atRow && comment.span > 1) {
        updateDataSelectedCount(atRow, isSelected)
        handledRows.set(atRow, atRow)

        const tbody = atRow.parentElement as HTMLElement
        let line = comment.lineNumberStart
        let rowAtStartLine: HTMLTableRowElement | null = null

        // Travel to get the first possible valid rowAtStartLine, as lines may
        // be collapsed or not rendered due to diff collapse/partial rendering
        while (!rowAtStartLine && line <= comment.lineNumberEnd) {
          rowAtStartLine = sideBySide
            ? (tbody.querySelector(`[data-annotation-for-line="${line}"]`)?.closest('tr') as HTMLTableRowElement)
            : (tbody
                .querySelector(`.line-num${comment.left ? 1 : 2}[data-line-number="${line}"]`)
                ?.closest('tr') as HTMLTableRowElement)
          line++
        }

        // When rowAtStartLine is found, mark all rows from it to atRow
        while (rowAtStartLine && rowAtStartLine !== atRow) {
          if (!handledRows.has(rowAtStartLine)) {
            updateDataSelectedCount(rowAtStartLine, isSelected)
            handledRows.set(rowAtStartLine, rowAtStartLine)
          }
          rowAtStartLine = rowAtStartLine.nextElementSibling as HTMLTableRowElement
        }
      }
    },
    [viewStyle, updateDataSelectedCount]
  )

  /**
   * Attach a single comment thread to its binding DOM. Thread can be a
   * draft/uncommitted/not-saved one.
   */
  const attachSingleCommentThread = useCallback(
    (commentThreadId: number, lineElements?: HTMLElement[]) => {
      const comment = comments.get(commentThreadId)

      // Early exit if conditions for the comment thread existing don't exist
      if (!comment || !isDiffRendered(contentRef)) return

      // If all comments in this thread are deleted, do nothing
      if (comment.commentItems.length && comment.commentItems.every(item => !!item.deleted)) return

      const isSideBySide = viewStyle === ViewStyle.SIDE_BY_SIDE
      const lineInfo = getCommentLineInfo(comment, contentRef.current, comment, viewStyle)

      if (!lineInfo.rowElement) return

      // Exit if comment thread is already rendered
      if (lineInfo.commentThreadRendered) {
        // Comment data changed, send updated data to CommentBox to update itself
        if (
          comment.commentItems?.[0]?.id > 0 &&
          comment.commentItems?.[0]?.id === comment._commentItems?.[0]?.id &&
          !isEqual(comment._commentItems, comment.commentItems)
        ) {
          comment._commentItems = structuredClone(comment.commentItems)
          dispatchCustomEvent(customEventForCommentWithId(commentThreadId), comment._commentItems)
        }
        return
      }

      // Add `selected` class into selected lines / rows (only if selected lines count > 1)
      markSelectedLines(comment, lineInfo.rowElement, true)

      lineInfo.rowElement.dataset.sourceLineNumber = String(comment.lineNumberEnd)

      // Annotate that the row is taken. We only support one thread per line as now
      updateDataCommentIds(lineInfo.rowElement, comment.inner.id as number, true)

      const isCommentThreadResolved = !!get(comment.commentItems[0], 'payload.resolved', false)

      // Create placeholder for opposite row (id < 0 means this is a new thread). This placeholder
      // expands itself when CommentBox's height is changed (in split view)
      const oppositeRowPlaceHolder = createCommentOppositePlaceHolder(comment.lineNumberEnd, commentThreadId < 0)

      // Attach the opposite placeholder (in split view)
      if (isSideBySide && !!lineInfo.oppositeRowElement) {
        lineInfo.oppositeRowElement.after(oppositeRowPlaceHolder)
      }

      // Create a new row below the lineNumber
      const commentRowElement = document.createElement('tr')
      commentRowElement.dataset.annotatedLine = String(comment.lineNumberEnd)
      commentRowElement.innerHTML = `<td colspan="2"></td>`
      lineInfo.rowElement.after(commentRowElement)

      // Set both place-holder and comment box hidden when comment thread is resolved
      if (isCommentThreadResolved) {
        oppositeRowPlaceHolder.setAttribute('hidden', '')
        commentRowElement.setAttribute('hidden', '')
      }

      commentRowElement.dataset.commentThreadStatus = isCommentThreadResolved
        ? CodeCommentState.RESOLVED
        : CodeCommentState.ACTIVE

      // `element` is where CommentBox will be mounted
      const element = commentRowElement.firstElementChild as HTMLTableCellElement

      // Attach a callback to `comment` to clean up CommentBox rendering and
      // reset states bound to lineInfo (called when Cancel button is clicked)
      comment.destroy = () => {
        ReactDOM.unmountComponentAtNode(element as HTMLDivElement)
        commentRowElement.parentElement?.removeChild(commentRowElement)
        lineInfo.oppositeRowElement?.parentElement?.removeChild(oppositeRowPlaceHolder as Element)
        updateDataCommentIds(lineInfo.rowElement, comment.inner.id as number, false)
        comment.destroy = undefined
        comments.delete(commentThreadId)

        // Update selected line on cancelling a comment thread
        if (lineElements?.length) {
          markSelectedLines(comment, lineInfo.rowElement, false)
        }
      }

      // Note: comment._commentItems keeps the latest data passed to CommentBox.
      // In case commentItems.commentItems is changed, we do a deep compare between
      // the two, if they are actually different, we send a signal to CommentBox to
      // update to the latest data
      comment._commentItems = structuredClone(comment.commentItems)

      const suggestionBlock = comment.left
        ? undefined
        : {
            source:
              comment.codeBlockContent ||
              (lineElements?.length
                ? lineElements
                    .map(td => td.nextElementSibling?.querySelector('.d2h-code-line-ctn')?.textContent)
                    .join('\n')
                : lineInfo.rowElement?.lastElementChild?.querySelector('.d2h-code-line-ctn')?.textContent || ''),
            lang: filenameToLanguage(diff.filePath.split('/').pop())
          }

      // Note: CommentBox is rendered as an independent React component.
      //       Everything passed to it must be either values, or refs.
      //       If you pass callbacks or states, they won't be updated and
      //       might cause unexpected bugs
      ReactDOM.unmountComponentAtNode(element as HTMLDivElement)
      ReactDOM.render(
        <AppWrapper>
          <CommentBox
            routingId={routingId}
            standalone={standalone}
            repoMetadata={repoMetadata}
            commentItems={comment._commentItems as CommentItem<TypesPullReqActivity>[]}
            initialContent={''}
            width={getCommentBoxWidth(isSideBySide)}
            onHeightChange={boxHeight => {
              const first = oppositeRowPlaceHolder?.firstElementChild as HTMLTableCellElement
              const last = oppositeRowPlaceHolder?.lastElementChild as HTMLTableCellElement

              if (first && last) {
                first.style.height = `${boxHeight}px`
                last.style.height = `${boxHeight}px`
              }
            }}
            autoFocusAndPosition={true}
            enableReplyPlaceHolder={(comment._commentItems as CommentItem<TypesPullReqActivity>[])?.length > 0}
            onCancel={comment.destroy}
            setDirty={setDirty || noop}
            currentUserName={currentUser?.display_name || currentUser?.email || ''}
            copyLinkToComment={copyLinkToComment}
            suggestionBlock={suggestionBlock}
            handleAction={async (action, value, commentItem) => {
              let result = true
              let updatedItem: CommentItem<TypesPullReqActivity> | undefined = undefined
              const id = (commentItem as CommentItem<TypesPullReqActivity>)?.id

              switch (action) {
                case CommentAction.NEW: {
                  const payload: OpenapiCommentCreatePullReqRequest = {
                    line_start: comment.lineNumberStart || comment.lineNumberEnd,
                    line_end: comment.lineNumberEnd,
                    line_start_new: !comment.left,
                    line_end_new: !comment.left,
                    path: diff.isRename && comment.left ? diff.oldName : diff.filePath,
                    source_commit_sha: sourceRef,
                    target_commit_sha: targetRef,
                    text: value
                  }

                  await save(payload)
                    .then((_activity: TypesPullReqActivity) => {
                      // Update data-comment-ids (replace tmp one with the new id coming from API)
                      updateDataCommentIds(lineInfo.rowElement, comment.inner.id as number, false)
                      updateDataCommentIds(lineInfo.rowElement, _activity.id as number, true)

                      updatedItem = activityToCommentItem(_activity)

                      // Delete the place-holder comment (negative id) in comments
                      comments.delete(commentThreadId)
                      comments.set(_activity.id as number, comment)
                    })
                    .catch(exception => {
                      result = false
                      showError(getErrorMessage(exception), 0)
                    })
                  break
                }

                case CommentAction.REPLY: {
                  await save({
                    type: CommentType.CODE_COMMENT,
                    text: value,
                    parent_id: Number(id)
                  })
                    .then((_activity: TypesPullReqActivity) => {
                      updatedItem = activityToCommentItem(_activity)
                    })
                    .catch(exception => {
                      result = false
                      showError(getErrorMessage(exception), 0)
                    })
                  break
                }

                case CommentAction.DELETE: {
                  result = false
                  await confirmAct({
                    message: getString('deleteCommentConfirm'),
                    action: async () => {
                      await remove({}, { pathParams: { id } })
                        .then(() => {
                          result = true
                        })
                        .catch(exception => {
                          result = false
                          showError(getErrorMessage(exception), 0)
                        })
                    }
                  })

                  if (result) {
                    // Reflect deleted item in comment.commentItems and comment._commentItems
                    const deletedItem1 = comment.commentItems.find(it => it.id === id)
                    const deletedItem2 = comment._commentItems?.find(it => it.id === id)

                    if (deletedItem1) deletedItem1.deleted = Date.now()
                    if (deletedItem2) deletedItem2.deleted = Date.now()

                    // Remove CommentBox if all items are deleted
                    if (comment._commentItems?.every(it => !!it.deleted)) {
                      // Remove `selected` class from selected lines / rows - if any
                      markSelectedLines(comment, lineInfo.rowElement, false)

                      // Destroy needs to be called outside of React life-cycle
                      setTimeout(comment.destroy || noop, 0)
                    }
                  }
                  break
                }

                case CommentAction.UPDATE: {
                  await update({ text: value }, { pathParams: { id } })
                    .then((_activity: TypesPullReqActivity) => {
                      updatedItem = activityToCommentItem(_activity)
                    })
                    .catch(exception => {
                      result = false
                      showError(getErrorMessage(exception), 0)
                    })
                  break
                }
              }

              // Trigger a manual activities fetch when action is UPDATE or DELETE since PR stats
              // won't change for these cases, causing activities not fetched automatically
              // TODO: Better to ask backend to send back an SSE event for these actions so UI
              // can handle in useGetPullRequestInfo instead of down here.
              if (result && (action === CommentAction.UPDATE || action === CommentAction.DELETE)) {
                refetchActivities?.()
              }

              return [result, updatedItem]
            }}
            outlets={{
              [CommentBoxOutletPosition.TOP]: (
                <CommentThreadTopDecoration startLine={comment.lineNumberStart} endLine={comment.lineNumberEnd} />
              ),
              [CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]: (
                <CodeCommentStatusSelect
                  repoMetadata={repoMetadata}
                  pullReqMetadata={pullReqMetadata as TypesPullReq}
                  comment={comment}
                  rowElement={commentRowElement}
                />
              ),
              [CommentBoxOutletPosition.RIGHT_OF_REPLY_PLACEHOLDER]: (
                <CodeCommentStatusButton
                  repoMetadata={repoMetadata}
                  pullReqMetadata={pullReqMetadata as TypesPullReq}
                  comment={comment}
                />
              ),
              [CommentBoxOutletPosition.BETWEEN_SAVE_AND_CANCEL_BUTTONS]: (props: ButtonProps) => (
                <CodeCommentSecondarySaveButton
                  repoMetadata={repoMetadata}
                  pullReqMetadata={pullReqMetadata as TypesPullReq}
                  comment={comment}
                  {...props}
                />
              )
            }}
          />
        </AppWrapper>,
        element
      )
    },
    [
      diff,
      comments,
      contentRef,
      viewStyle,
      routingId,
      standalone,
      repoMetadata,
      setDirty,
      currentUser?.display_name,
      currentUser?.email,
      pullReqMetadata,
      sourceRef,
      targetRef,
      save,
      showError,
      confirmAct,
      getString,
      remove,
      update,
      refetchActivities,
      copyLinkToComment,
      markSelectedLines,
      updateDataCommentIds
    ]
  )

  const selectoRef = useRef<Selecto>()

  const bindEventToSelectMultipleCodeLines = useCallback(
    () => {
      const containerDOM = containerRef.current

      if (containerDOM) {
        selectoRef.current?.destroy()

        selectoRef.current = new Selecto({
          className: css.selectoSelection,
          rootContainer: containerDOM,
          container: containerDOM,
          dragContainer: '.d2h-diff-table',
          dragCondition: e => {
            const classList = e.inputEvent?.srcElement?.classList

            // Allow drag to start on some certain elements only
            return (
              classList?.contains('d2h-code-line-prefix') ||
              classList?.contains('d2h-code-side-linenumber') ||
              classList?.contains('d2h-code-linenumber') ||
              classList?.contains('annotation-for-line') ||
              classList?.contains('line-num1') ||
              classList?.contains('line-num2')
            )
          },
          selectableTargets: [
            'td.d2h-code-side-linenumber:not(.d2h-info, .d2h-emptyplaceholder)',
            'td.d2h-code-linenumber:not(.d2h-info, .d2h-emptyplaceholder)',
            'td:not(.d2h-info, .d2h-emptyplaceholder) span[data-annotation-for-line]',
            'td:not(.d2h-info, .d2h-emptyplaceholder) .line-num1',
            'td:not(.d2h-info, .d2h-emptyplaceholder) .line-num2',
            'td:not(.d2h-info, .d2h-emptyplaceholder) .d2h-code-line-prefix'
          ],
          hitRate: 0,
          selectByClick: false,
          selectFromInside: true,
          ratio: 0,
          preventDragFromInside: false,
          preventClickEventOnDrag: true,
          preventDefault: false,
          clickBySelectEnd: true,
          continueSelect: false,
          continueSelectWithoutDeselect: true
        })

        selectoRef.current
          .on('select', ev => {
            ev.added.forEach(e => e.classList.add(selected))
            ev.removed.forEach(e => e.classList.remove(selected))
          })
          .on('selectEnd', ev => {
            // Selecto dispatches event to all instances wrongly, need to filter
            if (ev.added.length && ev.added[0]?.closest?.(`[data-diff-file-path="${diff.filePath}"]`)) {
              switch (ev.added.length) {
                case 1: {
                  ev.added.forEach(e => e.classList.remove(selected))
                  break
                }
                case 2: {
                  if (ev.added[1].classList.contains('d2h-code-line-prefix')) {
                    ev.added.forEach(e => e.classList.remove(selected))
                  }
                  break
                }
              }

              // Save current added elements
              const added = ev.added

              // Normalize added elements to make sure they are always unique line numbers
              ev.added = uniq(
                ev.added.map(e => {
                  if (e.classList.contains('d2h-code-line-prefix')) {
                    return e.closest?.('td')?.previousElementSibling as HTMLDivElement
                  } else if (e.classList.contains('annotation-for-line')) {
                    return e.closest?.('td') as HTMLDivElement
                  } else if (e.classList.contains('line-num1') || e.classList.contains('line-num2')) {
                    return e.closest?.('td') as HTMLDivElement
                  }

                  return e
                })
              )

              //
              // Notes:
              //
              // - Only allow selecting lines from a same hunk/chunk (continuous line numbers)
              // - Deselect all lines not belong to the last hunk
              //
              if (ev.added.length) {
                const selectedLineNumbers: number[] = []
                const _added: (HTMLElement | SVGElement)[] = []

                if (viewStyle === ViewStyle.LINE_BY_LINE) {
                  const lineNumbers1 = ev.added.map(e =>
                    parseInt(e.querySelector('.line-num1')?.textContent || '0', 10)
                  )
                  const lineNumbers2 = ev.added.map(e =>
                    parseInt(e.querySelector('.line-num2')?.textContent || '0', 10)
                  )
                  const lastLineNumberIndex1 = findLastIndex(lineNumbers1, ln => ln > 0)
                  const lastLineNumberIndex2 = findLastIndex(lineNumbers2, ln => ln > 0)

                  // Last line number in lineNumbers1 or lineNumbers2 determines the line numbers
                  // of which side will be picked
                  const lineNumbers =
                    lastLineNumberIndex2 >= lastLineNumberIndex1
                      ? lineNumbers2
                      : lastLineNumberIndex1 > lastLineNumberIndex2
                      ? lineNumbers1
                      : []

                  const len = lineNumbers.length
                  let index = len

                  while (--index >= 0) {
                    if (lineNumbers[index]) {
                      if (
                        index === len - 1 ||
                        lineNumbers[index] + 1 === lineNumbers[index + 1] ||
                        (lineNumbers[index] === 0 && lineNumbers[index - 1] + 1 === lineNumbers[index + 1]) ||
                        (lineNumbers[index + 1] === 0 && lineNumbers[index] + 1 === lineNumbers[index + 2])
                      ) {
                        _added.unshift(ev.added[index])
                        selectedLineNumbers.unshift(lineNumbers[index])
                      } else {
                        break
                      }
                    }
                  }
                } else {
                  const lineNumbers = ev.added.map(e => parseInt(e.textContent?.replace(/(\s)(\+)?/g, '') || '0', 10))
                  const len = lineNumbers.length
                  let index = len

                  while (--index >= 0) {
                    if (lineNumbers[index]) {
                      if (index === len - 1 || lineNumbers[index] + 1 === lineNumbers[index + 1]) {
                        _added.unshift(ev.added[index])
                        selectedLineNumbers.unshift(lineNumbers[index])
                      } else {
                        break
                      }
                    }
                  }
                }

                ev.added = _added

                // Remove selected class from child elements which Selecto picked
                added.forEach(e => e.classList.remove(selected))

                if (ev.added.length) {
                  startCommentThread(
                    { target: ev.added[ev.added.length - 1] } as unknown as MouseEvent,
                    ev.added as HTMLElement[],
                    selectedLineNumbers
                  )
                }
              }

              // Reset Selecto every time a selection is done
              selectoRef.current?.setSelectedTargets([])
            }
          })
      }
    },
    [containerRef.current, collapsed, diff] // eslint-disable-line react-hooks/exhaustive-deps
  )

  // Attach (render) all comment threads to their binding DOMs
  const attachAllCommentThreads = useCallback(() => {
    if (!readOnly) {
      if (isDiffRendered(contentRef) && comments.size > 0) {
        comments.forEach(item => attachSingleCommentThread(item.inner.id || 0))
      }

      bindEventToSelectMultipleCodeLines()
    }
  }, [readOnly, contentRef, comments, attachSingleCommentThread, bindEventToSelectMultipleCodeLines])

  // Detach (remove) all comment threads from their binding DOMs
  const detachAllCommentThreads = useCallback(() => {
    if (!readOnly && contentRef.current && isDiffRendered(contentRef)) {
      contentRef.current
        .querySelectorAll('[data-annotated-line]')
        .forEach(element => ReactDOM.unmountComponentAtNode(element.firstElementChild as HTMLTableCellElement))

      selectoRef.current?.destroy()
      selectoRef.current = undefined
    }
  }, [readOnly, contentRef])

  // Use a ref to keep latest comment items from activities to make sure rendering process is called
  // only when there's some real change from upstream (API) and not from React re-rendering process
  const diffCommentItemsRef = useRef<DiffCommentItem<TypesPullReqActivity>[]>()

  useEffect(
    function handleActivitiesChanged() {
      // Read only, no activities, commit range view, diff not rendered yet? Ignore handling comments
      if (readOnly || !activities?.length || (commitRange?.length || 0) > 0) {
        return
      }

      const _diffCommentItems = activitiesToDiffCommentItems(diff.filePath, activities)

      if (!isEqual(_diffCommentItems, diffCommentItemsRef.current)) {
        diffCommentItemsRef.current = _diffCommentItems

        _diffCommentItems.forEach(latestComment => {
          const { id } = latestComment.inner

          if (!id) return

          //
          // TODO: Re-evaluate below logics - we might not need them anymore!
          //

          const existingComment = comments.get(id)
          const latestDeleted = latestComment.commentItems.map(x => !!x.deleted).reduce((a, b) => a && b, true)

          // is this a to us new, but already delete comment? perfect, nothing to do
          if (!existingComment && latestDeleted) return

          // is this a new comment or we failed to render it before? add to our internal cache and render!
          if (!existingComment || !existingComment.destroy) {
            comments.set(id, latestComment)
            return
          }

          // heuristic: whoever has the latest update timestamp has the latest data
          const mostRecentExisting = existingComment.commentItems
            .map(x => max([x.updated as number, x.edited as number, x.deleted as number]) || 0)
            .reduce(
              (x, y) => max([x, y]) || 0,
              max([
                existingComment.inner.updated,
                existingComment.inner.edited,
                existingComment.inner.deleted as number,
                existingComment.inner.resolved
              ]) || 0
            )
          const mostRecentLatest = latestComment.commentItems
            .map(x => max([x.updated as number, x.edited as number, x.deleted as number]) || 0)
            .reduce(
              (x, y) => max([x, y]) || 0,
              max([
                latestComment.inner.updated,
                latestComment.inner.edited,
                latestComment.inner.deleted as number,
                latestComment.inner.resolved
              ]) || 0
            )

          if (mostRecentExisting >= mostRecentLatest) return

          // comment changed -- update everything that can change
          existingComment.inner = latestComment.inner
          existingComment.commentItems = [...latestComment.commentItems]
        })

        attachAllCommentThreads()
      }
    },
    [
      commitRange?.length,
      activities,
      diff.filePath,
      readOnly,
      comments,
      attachAllCommentThreads,
      containerRef,
      contentRef
    ]
  )

  useEffect(
    function handleCollapsedState() {
      if (!containerRef.current) return

      const containerDOM = containerRef.current as HTMLDivElement
      const { classList } = containerDOM

      if (collapsed) {
        classList.add(css.collapsed)

        // Fix scrolling position messes up with sticky header: When content of the diff content
        // is above the diff header, we need to scroll it back to below the header, adjust window
        // scrolling position to avoid the next diff scroll jump
        if (stickyTopPosition && containerDOM.getBoundingClientRect().y - stickyTopPosition < 1) {
          containerDOM.scrollIntoView()
          scrollElement.scroll({ top: (scrollElement.scrollTop || window.scrollY) - stickyTopPosition })
        }
      } else {
        classList.remove(css.collapsed)
      }
    },
    [readOnly, collapsed, stickyTopPosition, scrollElement, containerRef]
  )

  const startCommentThread = useCallback(
    function clickToAddAnnotation(event: MouseEvent, lineElements?: HTMLElement[], selectedLineNumbers?: number[]) {
      if (readOnly) return

      let target = event.target as HTMLDivElement
      const lineNumberCell = target?.closest('td.d2h-code-linenumber') || target?.closest('td.d2h-code-side-linenumber')

      // If click happens on line number, locate target from the line number as we allow
      // adding a comment by clicking line number as well
      if (lineNumberCell) {
        target = lineNumberCell.querySelector('[data-annotation-for-line]') as HTMLDivElement
      }

      const targetButton = target?.closest('[data-annotation-for-line]') as HTMLDivElement
      const annotatedLineRow = targetButton?.closest('tr') as HTMLTableRowElement

      if (targetButton && annotatedLineRow) {
        // If there's a new CommentBox showing, focus on it intead of creating a new one
        if (
          (annotatedLineRow.dataset.commentIds || '')
            .split('/')
            .map(Number)
            .find(_id => _id < 0)
        ) {
          let row = annotatedLineRow.nextElementSibling
          let editor: HTMLElement | null = null

          while (row && !editor) {
            editor = row.querySelector('[data-comment-thread-id=""] [contenteditable]')
            row = row.nextElementSibling
          }

          editor?.scrollIntoView({ block: 'center' })
          editor?.focus?.()

          return
        }

        // Utilize a random negative number as temporary IDs to prevent database entry collisions
        const randID = -(random(1_000_000, false) + random(1_000_000, false))
        const span = selectedLineNumbers?.length
          ? selectedLineNumbers[selectedLineNumbers.length - 1] - selectedLineNumbers[0] + 1
          : 0
        const commentItem: DiffCommentItem<TypesPullReqActivity> = {
          inner: { id: randID } as TypesPullReqActivity,
          left: false,
          right: false,
          lineNumberStart: 0,
          lineNumberEnd: 0,
          span: span > 1 ? span : 0,
          commentItems: [],
          filePath: '',
          destroy: undefined
        }

        if (viewStyle === ViewStyle.SIDE_BY_SIDE) {
          const leftParent = targetButton.closest('.d2h-file-side-diff.left')
          commentItem.left = !!leftParent
          commentItem.right = !leftParent
          commentItem.lineNumberEnd =
            selectedLineNumbers?.[selectedLineNumbers?.length - 1] || Number(targetButton.dataset.annotationForLine)
          commentItem.lineNumberStart = selectedLineNumbers?.[0] || commentItem.lineNumberEnd
        } else {
          const lineInfoTD = targetButton.closest('td')
          const lineNum1 = lineInfoTD?.querySelector('.line-num1')
          const lineNum2 = lineInfoTD?.querySelector('.line-num2')

          // Right has priority
          commentItem.right = !!lineNum2?.textContent
          commentItem.left = !commentItem.right
          commentItem.lineNumberEnd =
            selectedLineNumbers?.[selectedLineNumbers?.length - 1] ||
            Number(lineNum2?.textContent || lineNum1?.textContent)
          commentItem.lineNumberStart = selectedLineNumbers?.[0] || commentItem.lineNumberEnd
        }

        comments.set(randID, commentItem)

        attachSingleCommentThread(randID, lineElements)
      }
    },
    [viewStyle, readOnly, comments, attachSingleCommentThread]
  )

  const hookRef = useRef(
    useMemo(
      () => ({
        attachAllCommentThreads,
        detachAllCommentThreads
      }),
      [attachAllCommentThreads, detachAllCommentThreads]
    )
  )

  //
  // Detach all comment threads when component is unmounted
  // Note: Must use useLayoutEffect to ensure DOM references are still available
  //
  useLayoutEffect(() => () => hookRef.current?.detachAllCommentThreads(), [])

  useEffect(
    function bindClickEventToStartNewCommentThread() {
      const click = 'click'
      const containerDOM = containerRef.current

      // Note: Can't use useEventListener() as containerDOM can be null
      if (containerDOM) {
        containerDOM.addEventListener(click, startCommentThread)
        return () => containerDOM.removeEventListener(click, startCommentThread)
      }
    },
    [containerRef.current] // eslint-disable-line react-hooks/exhaustive-deps
  )

  // To avoid multiple re-rendering cycles from DiffViewer
  // component, return a ref instead of an object
  return hookRef
}

/**
 * Decide CommentBox container width.
 */
function getCommentBoxWidth(isSideBySide: boolean) {
  return isSideBySide ? `min(calc(var(--page-container-width) - 48px)/2, 900px)` : undefined
}

/**
 * Hook to provide comment actions to interact with APIs.
 */
function useCommentAPI(path: string) {
  const { mutate: save } = useMutate({ verb: 'POST', path })
  const { mutate: update } = useMutate({ verb: 'PATCH', path: ({ id }) => `${path}/${id}` })
  const { mutate: remove } = useMutate({ verb: 'DELETE', path: ({ id }) => `${path}/${id}` })

  return { save, update, remove }
}

/**
 * Test if diff is rendered. The condition to check is DiffViewer renders a single element
 * with [data] attribute or a class `d2h-wrapper` (in case DiffViewer changes structure in future versions).
 */
function isDiffRendered(ref: React.RefObject<HTMLDivElement | null>) {
  return !!ref.current?.querySelector('[data]' || !!ref.current?.querySelector('.d2h-wrapper'))
}

const selected = 'selected'
