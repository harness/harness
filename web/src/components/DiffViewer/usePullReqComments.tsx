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
import ReactDOM from 'react-dom'
import { useToaster, ButtonProps, useIsMounted } from '@harnessio/uicore'
import { isEqual, max, noop, random } from 'lodash-es'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import type { DiffFileEntry } from 'utils/types'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useAppContext } from 'AppContext'
import type { OpenapiCommentCreatePullReqRequest, TypesPullReq, TypesPullReqActivity } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { AppWrapper } from 'App'
import { CodeCommentStatusButton } from 'components/CodeCommentStatusButton/CodeCommentStatusButton'
import { CodeCommentSecondarySaveButton } from 'components/CodeCommentSecondarySaveButton/CodeCommentSecondarySaveButton'
import { CodeCommentStatusSelect } from 'components/CodeCommentStatusSelect/CodeCommentStatusSelect'
import { useQueryParams } from 'hooks/useQueryParams'
import { dispatchCustomEvent, useEventListener } from 'hooks/useEventListener'
import { UseGetPullRequestInfoResult, usePullReqActivities } from 'pages/PullRequest/useGetPullRequestInfo'
import {
  activitiesToDiffCommentItems,
  activityToCommentItem,
  CommentType,
  DiffCommentItem,
  DIFF_VIEWER_HEADER_HEIGHT,
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
import { DiffViewerCustomEvent, DiffViewerEvent } from './DiffViewer'
import css from './DiffViewer.module.scss'

interface UsePullReqCommentsProps extends Pick<GitInfoProps, 'repoMetadata'> {
  diffs: DiffFileEntry[]
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
  diffs,
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
  const { routingId, currentUser, standalone } = useAppContext()
  const { showError } = useToaster()
  const confirmAct = useConfirmAct()
  const commentPath = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata?.number}/comments`,
    [repoMetadata.path, pullReqMetadata?.number]
  )
  const isMounted = useIsMounted()
  const { path, commentId } = useQueryParams<{ path: string; commentId: string }>()
  const { save, update, remove } = useCommentAPI(commentPath)
  const [comments] = useState(new Map<number, DiffCommentItem<TypesPullReqActivity>>())

  //
  // Attach a single comment thread to its binding DOM. Thread can be a
  // draft/uncommitted/not-saved one.
  //
  const attachSingleCommentThread = useCallback(
    (commentThreadId: number) => {
      const comment = comments.get(commentThreadId)

      // Early exit if conditions for the comment thread existing don't exist
      if (!comment || !isDiffRendered(contentRef)) return

      // If all comments in this thread are deleted, do nothing
      if (comment.commentItems.length && comment.commentItems.every(item => !!item.deleted)) return

      const isSideBySide = viewStyle === ViewStyle.SIDE_BY_SIDE
      const lineInfo = getCommentLineInfo(contentRef.current, comment, viewStyle)

      // Early exit if rowElement does not exist, or already a thread rendered
      // TODO: Support adding multiple comments on a same line
      if (!lineInfo.rowElement || lineInfo.hasCommentsRendered) {
        // Comment data changed, send updated data to CommentBox to update itself
        if (lineInfo.hasCommentsRendered && !isEqual(comment._commentItems, comment.commentItems)) {
          comment._commentItems = structuredClone(comment.commentItems)
          dispatchCustomEvent(customEventForCommentWithId(commentThreadId), comment._commentItems)
        }
        return
      }

      // Annotate that the row is taken. We only support one thread per line as now
      lineInfo.rowElement.dataset.annotated = 'true'

      // Create placeholder for opposite row (id < 0 means this is a new thread). This placeholder
      // expands itself when CommentBox's height is changed (in split view)
      const oppositeRowPlaceHolder = createCommentOppositePlaceHolder(comment.lineNumber, commentThreadId < 0)

      // Attach the opposite placeholder (in split view)
      if (isSideBySide && !!lineInfo.oppositeRowElement) {
        lineInfo.oppositeRowElement.after(oppositeRowPlaceHolder)
      }

      // Create a new row below the lineNumber
      const commentRowElement = document.createElement('tr')
      commentRowElement.dataset.annotatedLine = String(comment.lineNumber)
      commentRowElement.innerHTML = `<td colspan="2"></td>`
      lineInfo.rowElement.after(commentRowElement)

      // `element` is where CommentBox will be mounted
      const element = commentRowElement.firstElementChild as HTMLTableCellElement

      // Attach a callback to `comment` to clean up CommentBox rendering and
      // reset states bound to lineInfo (called when Cancel button is clicked)
      comment.destroy = () => {
        ReactDOM.unmountComponentAtNode(element as HTMLDivElement)
        commentRowElement.parentElement?.removeChild(commentRowElement)
        lineInfo.oppositeRowElement?.parentElement?.removeChild(oppositeRowPlaceHolder as Element)
        delete lineInfo.rowElement.dataset.annotated
        comment.destroy = undefined
        comments.delete(commentThreadId)
      }

      // Note: comment._commentItems keeps the latest data passed to CommentBox.
      // In case commentItems.commentItems is changed, we do a deep compare between
      // the two, if they are actually different, we send a signal to CommentBox to
      // update to the latest data
      comment._commentItems = structuredClone(comment.commentItems)

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
            handleAction={async (action, value, commentItem) => {
              let result = true
              let updatedItem: CommentItem<TypesPullReqActivity> | undefined = undefined
              const id = (commentItem as CommentItem<TypesPullReqActivity>)?.id

              switch (action) {
                case CommentAction.NEW: {
                  const payload: OpenapiCommentCreatePullReqRequest = {
                    line_start: comment.lineNumber,
                    line_end: comment.lineNumber,
                    line_start_new: !comment.left,
                    line_end_new: !comment.left,
                    path: diff.isRename && comment.left ? diff.oldName : diff.filePath,
                    source_commit_sha: sourceRef,
                    target_commit_sha: targetRef,
                    text: value
                  }

                  await save(payload)
                    .then((_activity: TypesPullReqActivity) => {
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

                  // Reflect deleted item in comment.commentItems and comment._commentItems
                  const deletedItem1 = comment.commentItems.find(it => it.id === id)
                  const deletedItem2 = comment._commentItems?.find(it => it.id === id)

                  if (deletedItem1) deletedItem1.deleted = Date.now()
                  if (deletedItem2) deletedItem2.deleted = Date.now()

                  // Remove CommentBox if all items are deleted
                  if (comment._commentItems?.every(it => !!it.deleted)) {
                    setTimeout(comment.destroy || noop, 0)
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
              [CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]: (
                <CodeCommentStatusSelect
                  repoMetadata={repoMetadata}
                  pullReqMetadata={pullReqMetadata as TypesPullReq}
                  commentItems={comment._commentItems as CommentItem<TypesPullReqActivity>[]}
                />
              ),
              [CommentBoxOutletPosition.RIGHT_OF_REPLY_PLACEHOLDER]: (
                <CodeCommentStatusButton
                  repoMetadata={repoMetadata}
                  pullReqMetadata={pullReqMetadata as TypesPullReq}
                  commentItems={comment._commentItems as CommentItem<TypesPullReqActivity>[]}
                />
              ),
              [CommentBoxOutletPosition.BETWEEN_SAVE_AND_CANCEL_BUTTONS]: (props: ButtonProps) => (
                <CodeCommentSecondarySaveButton
                  repoMetadata={repoMetadata}
                  pullReqMetadata={pullReqMetadata as TypesPullReq}
                  commentItems={comment._commentItems as CommentItem<TypesPullReqActivity>[]}
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
      comments,
      confirmAct,
      contentRef,
      currentUser?.display_name,
      currentUser?.email,
      diff.filePath,
      getString,
      pullReqMetadata,
      repoMetadata,
      routingId,
      save,
      update,
      remove,
      showError,
      sourceRef,
      standalone,
      targetRef,
      viewStyle,
      refetchActivities,
      setDirty
    ]
  )

  // Attach (render) all comment threads to their binding DOMs
  const attachAllCommentThreads = useCallback(() => {
    if (!readOnly && isDiffRendered(contentRef)) {
      comments.forEach(item => attachSingleCommentThread(item.inner.id || 0))
    }
  }, [readOnly, contentRef, comments, attachSingleCommentThread])

  // Detach (remove) all comment threads from their binding DOMs
  const detachAllCommentThreads = useCallback(() => {
    if (!readOnly && contentRef.current && isDiffRendered(contentRef)) {
      contentRef.current
        .querySelectorAll('[data-annotated-line]')
        .forEach(element => ReactDOM.unmountComponentAtNode(element.firstElementChild as HTMLTableCellElement))
    }
  }, [readOnly, contentRef])

  // Use a ref to keep latest comment items from activities to make sure rendering process is called
  // only when there's some real change from upstream (API) and not from React re-rendering process
  const diffCommentItemsRef = useRef<DiffCommentItem<TypesPullReqActivity>[]>()

  useEffect(
    function handleActivitiesChanged() {
      // Read only, no activities, commit range view, diff not rendered yet? Ignore handling comments
      if (readOnly || !activities?.length || (commitRange?.length || 0) > 0 || !isDiffRendered(contentRef)) {
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
      const { classList, style } = containerDOM

      if (collapsed) {
        classList.add(css.collapsed)

        if (parseInt(style.height) != DIFF_VIEWER_HEADER_HEIGHT) {
          style.height = `${DIFF_VIEWER_HEADER_HEIGHT}px`
        }

        // Fix scrolling position messes up with sticky header: When content of the diff content
        // is above the diff header, we need to scroll it back to below the header, adjust window
        // scrolling position to avoid the next diff scroll jump
        if (stickyTopPosition && containerDOM.getBoundingClientRect().y - stickyTopPosition < 1) {
          containerDOM.scrollIntoView()
          scrollElement.scroll({ top: (scrollElement.scrollTop || window.scrollY) - stickyTopPosition })
        }
      } else {
        classList.remove(css.collapsed)

        const newHeight = Number(containerDOM.scrollHeight)

        if (parseInt(style.height) != newHeight) {
          style.height = `${newHeight}px`
        }
      }
    },
    [readOnly, collapsed, stickyTopPosition, scrollElement, containerRef]
  )

  const startCommentThread = useCallback(
    function clickToAddAnnotation(event: MouseEvent) {
      if (readOnly) return

      const target = event.target as HTMLDivElement
      const targetButton = target?.closest('[data-annotation-for-line]') as HTMLDivElement
      const annotatedLineRow = targetButton?.closest('tr') as HTMLTableRowElement

      // Utilize a random negative number as temporary IDs to prevent database entry collisions
      const randID = -(random(1_000_000, false) + random(1_000_000, false))
      const commentItem: DiffCommentItem<TypesPullReqActivity> = {
        inner: { id: randID } as TypesPullReqActivity,
        left: false,
        right: false,
        lineNumber: 0,
        commentItems: [],
        filePath: '',
        destroy: undefined
      }

      if (targetButton && annotatedLineRow) {
        if (viewStyle === ViewStyle.SIDE_BY_SIDE) {
          const leftParent = targetButton.closest('.d2h-file-side-diff.left')
          commentItem.left = !!leftParent
          commentItem.right = !leftParent
          commentItem.lineNumber = Number(targetButton.dataset.annotationForLine)
        } else {
          const lineInfoTD = targetButton.closest('td')?.previousElementSibling
          const lineNum1 = lineInfoTD?.querySelector('.line-num1')
          const lineNum2 = lineInfoTD?.querySelector('.line-num2')

          // Right has priority
          commentItem.right = !!lineNum2?.textContent
          commentItem.left = !commentItem.right
          commentItem.lineNumber = Number(lineNum2?.textContent || lineNum1?.textContent)
        }

        comments.set(randID, commentItem)

        attachSingleCommentThread(randID)
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

  //
  // Scroll into view if `path` query param matched with diff.filePath.
  //
  useEffect(() => {
    if (readOnly || !path || !commentId || !containerRef.current) {
      return
    }

    if (path === diff.filePath) {
      const index = diffs.findIndex(_diff => _diff.filePath === diff.filePath)

      if (index >= 0) {
        dispatchCustomEvent<DiffViewerCustomEvent>(diff.filePath, {
          action: DiffViewerEvent.SCROLL_INTO_VIEW,
          diffs,
          index,
          commentId,
          onDone: noop
        })
      }
    }
  }, [diffs, readOnly, path, commentId, diff.filePath, isMounted, containerRef])

  //
  // Add click event listener to start a new comment thread
  //
  useEventListener('click', startCommentThread, containerRef.current as HTMLDivElement)

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
