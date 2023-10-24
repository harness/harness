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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useMutate } from 'restful-react'
import ReactDOM from 'react-dom'
import { useInView } from 'react-intersection-observer'
import {
  Button,
  Container,
  FlexExpander,
  ButtonVariation,
  Layout,
  Text,
  ButtonSize,
  useToaster,
  ButtonProps,
  Checkbox,
  useIsMounted
} from '@harnessio/uicore'
import cx from 'classnames'
import { Render } from 'react-jsx-match'
import { Link } from 'react-router-dom'
import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import { max, random } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { useEventListener } from 'hooks/useEventListener'
import type { DiffFileEntry } from 'utils/types'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useAppContext } from 'AppContext'
import type { OpenapiCommentCreatePullReqRequest, TypesPullReq, TypesPullReqActivity } from 'services/code'
import { getErrorMessage, waitUntil } from 'utils/Utils'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { AppWrapper } from 'App'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import { CodeCommentStatusButton } from 'components/CodeCommentStatusButton/CodeCommentStatusButton'
import { CodeCommentSecondarySaveButton } from 'components/CodeCommentSecondarySaveButton/CodeCommentSecondarySaveButton'
import { CodeCommentStatusSelect } from 'components/CodeCommentStatusSelect/CodeCommentStatusSelect'
import { useQueryParams } from 'hooks/useQueryParams'
import {
  activitiesToDiffCommentItems,
  activityToCommentItem,
  CommentType,
  DIFF2HTML_CONFIG,
  DiffCommentItem,
  DIFF_VIEWER_HEADER_HEIGHT,
  getCommentLineInfo,
  createCommentOppositePlaceHolder,
  ViewStyle,
  contentDOMHasData
} from './DiffViewerUtils'
import {
  CommentAction,
  CommentBox,
  CommentBoxOutletPosition,
  CommentItem,
  SingleConsumerEventStream
} from '../CommentBox/CommentBox'
import css from './DiffViewer.module.scss'

interface DiffViewerProps extends Pick<GitInfoProps, 'repoMetadata'> {
  diff: DiffFileEntry
  viewStyle: ViewStyle
  stickyTopPosition?: number
  readOnly?: boolean
  pullRequestMetadata?: TypesPullReq
  onCommentUpdate: () => void
  targetRef?: string
  sourceRef?: string
  commitRange?: string[]
  scrollElement: HTMLElement
}

//
// Note: Lots of direct DOM manipulations are used to boost performance.
//       Avoid React re-rendering at all cost as it might cause unresponsive UI
//       when diff content is big, or when a PR has a lot of changed files.
//
export const DiffViewer: React.FC<DiffViewerProps> = ({
  diff,
  viewStyle,
  stickyTopPosition = 0,
  readOnly,
  repoMetadata,
  pullRequestMetadata,
  onCommentUpdate,
  targetRef,
  sourceRef,
  commitRange,
  scrollElement
}) => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const viewedPath = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/file-views`,
    [repoMetadata.path, pullRequestMetadata?.number]
  )
  const { mutate: markViewed } = useMutate({ verb: 'PUT', path: viewedPath })
  const { mutate: unmarkViewed } = useMutate({ verb: 'DELETE', path: ({ filePath }) => `${viewedPath}/${filePath}` })

  // file viewed feature is only enabled if no commit range is provided (otherwise component is hidden, too)
  const [viewed, setViewed] = useState(
    commitRange?.length === 0 && diff.fileViews?.get(diff.filePath) === diff.checksumAfter
  )
  useEffect(() => {
    if (commitRange?.length === 0) {
      setViewed(diff.fileViews?.get(diff.filePath) === diff.checksumAfter)
    }
  }, [diff.fileViews, diff.filePath, diff.checksumAfter, commitRange])

  const [collapsed, setCollapsed] = useState(viewed)
  useEffect(() => {
    setCollapsed(viewed)
  }, [viewed])
  const [fileUnchanged] = useState(diff.unchangedPercentage === 100)
  const [fileDeleted] = useState(diff.isDeleted)
  const [renderCustomContent, setRenderCustomContent] = useState(fileUnchanged || fileDeleted)
  const [diffRenderer, setDiffRenderer] = useState<Diff2HtmlUI>()
  const { ref: inViewRef, inView } = useInView({ rootMargin: '100px 0px' })
  const containerRef = useRef<HTMLDivElement | null>(null)
  const { currentUser, standalone } = useAppContext()
  const { showError } = useToaster()
  const confirmAct = useConfirmAct()
  const commentPath = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/comments`,
    [repoMetadata.path, pullRequestMetadata?.number]
  )
  const { mutate: saveComment } = useMutate({ verb: 'POST', path: commentPath })
  const { mutate: updateComment } = useMutate({ verb: 'PATCH', path: ({ id }) => `${commentPath}/${id}` })
  const { mutate: deleteComment } = useMutate({ verb: 'DELETE', path: ({ id }) => `${commentPath}/${id}` })

  const [comments] = useState(new Map<number, DiffCommentItem<TypesPullReqActivity>>())

  const [dirty, setDirty] = useState(false)
  const setContainerRef = useCallback(
    node => {
      containerRef.current = node
      inViewRef(node)
    },
    [inViewRef]
  )
  const contentRef = useRef<HTMLDivElement>(null)
  const setupViewerInitialStates = useCallback(() => {
    setDiffRenderer(
      new Diff2HtmlUI(
        document.getElementById(diff.contentId) as HTMLElement,
        [diff],
        Object.assign({}, DIFF2HTML_CONFIG, { outputFormat: viewStyle })
      )
    )
  }, [diff, viewStyle])

  // renderCodeComment renders a single comment (both newly created ones and existing ones frm the db!)
  const renderCodeComment = useCallback(
    (id: number) => {
      const comment = comments.get(id)
      if (!comment || comment.destroy) {
        return
      }

      // early exit if there's nothing to render on
      if (!contentRef.current || !contentDOMHasData(contentRef.current)) {
        return
      }

      const isSideBySide = viewStyle === ViewStyle.SIDE_BY_SIDE

      const lineInfo = getCommentLineInfo(contentRef.current, comment, viewStyle)

      // TODO: add support for live updating changes and replies to comment!
      if (!lineInfo.rowElement || lineInfo.hasCommentsRendered) {
        return
      }
      const { rowElement } = lineInfo

      // Annotate row to indicated the row is taken (we only support one per line as of now)
      rowElement.dataset.annotated = 'true'

      // always create placeholder (in memory)
      const oppositeRowPlaceHolder = createCommentOppositePlaceHolder(comment.lineNumber)

      // in split view, actually attach the placeholder
      if (isSideBySide && lineInfo.oppositeRowElement != null) {
        lineInfo.oppositeRowElement.after(oppositeRowPlaceHolder)
      }

      // Create a new row below it and render CommentBox inside
      const commentRowElement = document.createElement('tr')
      commentRowElement.dataset.annotatedLine = String(comment.lineNumber)
      commentRowElement.innerHTML = `<td colspan="2"></td>`
      rowElement.after(commentRowElement)

      const element = commentRowElement.firstElementChild as HTMLTableCellElement
      comment.destroy = () => {
        // Clean up CommentBox rendering and reset states bound to lineInfo
        ReactDOM.unmountComponentAtNode(element as HTMLDivElement)
        commentRowElement.parentElement?.removeChild(commentRowElement)
        lineInfo.oppositeRowElement?.parentElement?.removeChild(oppositeRowPlaceHolder as Element)
        delete lineInfo.rowElement.dataset.annotated
        comment.destroy = undefined
        comments.delete(id)
      }

      comment.eventStream = new SingleConsumerEventStream<CommentItem<TypesPullReqActivity>[]>()

      // Note: CommentBox is rendered as an independent React component
      //       everything passed to it must be either values, or refs. If you
      //       pass callbacks or states, they won't be updated and might
      //       cause unexpected bugs
      ReactDOM.unmountComponentAtNode(element as HTMLDivElement)
      ReactDOM.render(
        <AppWrapper>
          <CommentBox
            commentItems={comment.commentItems}
            eventStream={comment.eventStream}
            initialContent={''}
            width={isSideBySide ? 'calc(50vw - 200px)' : undefined}
            onHeightChange={boxHeight => {
              const first = oppositeRowPlaceHolder?.firstElementChild as HTMLTableCellElement
              const last = oppositeRowPlaceHolder?.lastElementChild as HTMLTableCellElement
              if (first && last) {
                first.style.height = `${boxHeight}px`
                last.style.height = `${boxHeight}px`
              }
            }}
            autoFocusAndPosition={true}
            enableReplyPlaceHolder={comment.commentItems?.length > 0}
            onCancel={comment.destroy}
            setDirty={setDirty}
            currentUserName={currentUser.display_name}
            handleAction={async (action, value, commentItem) => {
              let result = true
              let updatedItem: CommentItem<TypesPullReqActivity> | undefined = undefined
              const existingDBID = (commentItem as CommentItem<TypesPullReqActivity>)?.id

              switch (action) {
                case CommentAction.NEW: {
                  const payload: OpenapiCommentCreatePullReqRequest = {
                    line_start: comment.lineNumber,
                    line_end: comment.lineNumber,
                    line_start_new: !comment.left,
                    line_end_new: !comment.left,
                    path: diff.filePath,
                    source_commit_sha: sourceRef,
                    target_commit_sha: targetRef,
                    text: value
                  }

                  await saveComment(payload)
                    .then((createdActivity: TypesPullReqActivity) => {
                      if (!createdActivity.id) {
                        comment.destroy?.()
                        return
                      }

                      updatedItem = activityToCommentItem(createdActivity)

                      // recreate comment for now to tie any loose ends
                      comment.destroy?.()
                      comments.delete(id)
                      comments.set(createdActivity.id, comment)

                      // persist comment in activities
                      diff.activities?.push(createdActivity)
                    })
                    .catch(exception => {
                      result = false
                      showError(getErrorMessage(exception), 0)
                    })
                  break
                }

                case CommentAction.REPLY: {
                  await saveComment({
                    type: CommentType.CODE_COMMENT,
                    text: value,
                    parent_id: Number(existingDBID)
                  })
                    .then(createdActivity => {
                      if (!createdActivity.id) {
                        comment.destroy?.()
                        return
                      }

                      updatedItem = activityToCommentItem(createdActivity)

                      // persist comment in activities - will update parent comment with latest replies
                      diff.activities?.push(createdActivity)
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
                      await deleteComment({}, { pathParams: { id: existingDBID } })
                        .then(() => {
                          result = true
                        })
                        .catch(exception => {
                          result = false
                          showError(getErrorMessage(exception), 0, getString('pr.failedToDeleteComment'))
                        })
                    }
                  })
                  break
                }

                case CommentAction.UPDATE: {
                  await updateComment({ text: value }, { pathParams: { id: existingDBID } })
                    .then(newComment => {
                      updatedItem = activityToCommentItem(newComment)
                    })
                    .catch(exception => {
                      result = false
                      showError(getErrorMessage(exception), 0)
                    })
                  break
                }
              }

              if (result) {
                onCommentUpdate()
              }

              return [result, updatedItem]
            }}
            outlets={{
              [CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]: (
                <CodeCommentStatusSelect
                  repoMetadata={repoMetadata}
                  pullRequestMetadata={pullRequestMetadata as TypesPullReq}
                  onCommentUpdate={onCommentUpdate}
                  commentItems={comment.commentItems}
                />
              ),
              [CommentBoxOutletPosition.LEFT_OF_REPLY_PLACEHOLDER]: (
                <CodeCommentStatusButton
                  repoMetadata={repoMetadata}
                  pullRequestMetadata={pullRequestMetadata as TypesPullReq}
                  onCommentUpdate={onCommentUpdate}
                  commentItems={comment.commentItems}
                />
              ),
              [CommentBoxOutletPosition.BETWEEN_SAVE_AND_CANCEL_BUTTONS]: (props: ButtonProps) => (
                <CodeCommentSecondarySaveButton
                  repoMetadata={repoMetadata}
                  pullRequestMetadata={pullRequestMetadata as TypesPullReq}
                  commentItems={comment.commentItems}
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
      // eslint-disable-line react-hooks/exhaustive-deps
      comments,
      currentUser.display_name,
      deleteComment,
      diff.activities,
      diff.filePath,
      pullRequestMetadata,
      repoMetadata,
      saveComment,
      showError,
      sourceRef,
      targetRef,
      updateComment,
      viewStyle

      // The following three cause the whole thing to re-render and run unnecessary all the time.
      // disable for now as no issues were found in testing.
      // getString,
      // onCommentUpdate,
      // confirmAct,
    ]
  )

  // reRenderCodeComments is required to trigger comment rendering once all data is the renderer completed drawing..
  const reRenderCodeComments = useCallback(() => {
    if (readOnly) {
      return
    }

    // early exit if there's nothing to render on
    if (!contentRef.current || !contentDOMHasData(contentRef.current)) {
      return
    }

    comments.forEach(item => renderCodeComment(item.inner.id || 0))
  }, [readOnly, comments, renderCodeComment])

  useEffect(function cleanUpCommentBoxRendering() {
    const contentDOM = contentRef.current
    return () => {
      contentDOM
        ?.querySelectorAll('[data-annotated-line]')
        .forEach(element => ReactDOM.unmountComponentAtNode(element.firstElementChild as HTMLTableCellElement))
    }
  }, [])

  useEffect(
    function handleActivityChanges() {
      // no activities or commit range view? no comments!
      if (!diff?.activities || (commitRange?.length || 0) > 0) {
        return
      }

      const latestComments = activitiesToDiffCommentItems(diff.filePath, diff.activities)

      latestComments.forEach(latestComment => {
        const id = latestComment.inner.id
        if (!id) {
          return
        }

        const existingComment = comments.get(id)
        const latestDeleted = latestComment.commentItems.map(x => !!x.deleted).reduce((a, b) => a && b, true)

        // is this a to us new, but already delete comment? perfect, nothing to do
        if (!existingComment && latestDeleted) {
          return
        }

        // is this a new comment or we failed to render it before? add to our internal cache and render!
        if (!existingComment || !existingComment.destroy) {
          comments.set(id, latestComment)
          renderCodeComment(id)
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
        if (mostRecentExisting >= mostRecentLatest) {
          return
        }

        // comment changed -- update everything that can change
        existingComment.inner = latestComment.inner
        existingComment.commentItems = [...latestComment.commentItems]

        // push event to subscriber
        existingComment.eventStream?.publish(existingComment.commentItems)

        // NOTE: no need to render comment as we update in place :)
      })
    },
    [diff.filePath, diff?.activities, diff?.activities?.length, commitRange, comments, renderCodeComment]
  )

  useEffect(
    function createDiffRenderer() {
      if (inView && !diffRenderer) {
        setupViewerInitialStates()
      }
    },
    [inView, diffRenderer, setupViewerInitialStates]
  )

  useEffect(
    function handleCollapsedState() {
      const containerDOM = containerRef.current as HTMLDivElement & { scrollIntoViewIfNeeded: () => void }
      const { classList: containerClassList, style: containerStyle } = containerDOM

      if (collapsed) {
        containerClassList.add(css.collapsed)

        if (parseInt(containerStyle.height) != DIFF_VIEWER_HEADER_HEIGHT) {
          containerStyle.height = `${DIFF_VIEWER_HEADER_HEIGHT}px`
        }

        // Fix scrolling position messes up with sticky header: When content of the diff content
        // is above the diff header, we need to scroll it back to below the header, adjust window
        // scrolling position to avoid the next diff scroll jump
        const { y } = containerDOM.getBoundingClientRect()
        if (y - stickyTopPosition < 1) {
          containerDOM.scrollIntoView()

          if (stickyTopPosition) {
            scrollElement.scroll({ top: scrollElement.scrollTop - stickyTopPosition })
          }
        }
      } else {
        containerClassList.remove(css.collapsed)

        const newHeight = Number(containerDOM.scrollHeight)
        if (parseInt(containerStyle.height) != newHeight) {
          containerStyle.height = `${newHeight}px`
        }
      }
    },
    [collapsed, stickyTopPosition, scrollElement]
  )

  useEventListener(
    'click',
    useCallback(
      function clickToAddAnnotation(event: MouseEvent) {
        if (readOnly) {
          return
        }

        const target = event.target as HTMLDivElement
        const targetButton = target?.closest('[data-annotation-for-line]') as HTMLDivElement
        const annotatedLineRow = targetButton?.closest('tr') as HTMLTableRowElement

        // use random negative numbers as temporary IDs - never collides with persisted db entries :P
        const randID = -random(1000000000, false)
        const commentItem: DiffCommentItem<TypesPullReqActivity> = {
          inner: { id: randID } as TypesPullReqActivity,
          left: false,
          right: false,
          lineNumber: 0,
          commentItems: [],
          filePath: '',
          destroy: undefined,
          eventStream: undefined
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

          renderCodeComment(randID)
        }
      },
      [viewStyle, readOnly, comments, renderCodeComment]
    ),
    containerRef.current as HTMLDivElement
  )

  const renderDiffAndUpdateContainerHeightIfNeeded = useCallback(
    (enforced = false) => {
      const contentDOM = contentRef.current as HTMLDivElement
      const containerDOM = containerRef.current as HTMLDivElement

      if (!contentDOM.dataset.rendered || enforced) {
        if (!renderCustomContent || enforced) {
          containerDOM.style.height = 'auto'
          diffRenderer?.draw()
          reRenderCodeComments()
        }

        contentDOM.dataset.rendered = 'true'
      }
    },
    [diffRenderer, renderCustomContent, reRenderCodeComments]
  )

  useEffect(
    function renderInitialContent() {
      if (diffRenderer && inView) {
        renderDiffAndUpdateContainerHeightIfNeeded()
      }
    },
    [inView, diffRenderer, renderDiffAndUpdateContainerHeightIfNeeded]
  )

  const isMounted = useIsMounted()
  const { path, commentId } = useQueryParams<{ path: string; commentId: string }>()

  useEffect(
    function scrollToComment() {
      if (path && commentId && path === diff.filePath) {
        containerRef.current?.scrollIntoView({ block: 'start' })

        waitUntil(
          () => !!containerRef.current?.querySelector(`[data-comment-id="${commentId}"]`),
          () => {
            const dom = containerRef.current?.querySelector(`[data-comment-id="${commentId}"]`)?.parentElement
              ?.parentElement?.parentElement?.parentElement

            if (dom) {
              window.requestAnimationFrame(() => {
                setTimeout(() => {
                  if (isMounted.current) {
                    dom?.scrollIntoView({ block: 'center' })
                  }
                }, 500)
              })
            }
          }
        )
      }
    },
    [path, commentId]
  )
  const [isSidebarExpanded, setIsSidebarExpanded] = useState(true)
  const sideBarExpandedHandler = useCallback((event: Event) => {
    setIsSidebarExpanded(_ => !!(event as CustomEvent).detail)
  }, [])

  useEffect(() => {
    window.addEventListener(SIDE_NAV_EXPANDED_EVENT, sideBarExpandedHandler)
    return () => window.removeEventListener(SIDE_NAV_EXPANDED_EVENT, sideBarExpandedHandler)
  }, [sideBarExpandedHandler])

  return (
    <Container
      ref={setContainerRef}
      id={diff.containerId}
      className={cx(css.main, { [css.readOnly]: readOnly })}
      style={{ '--diff-viewer-sticky-top': `${stickyTopPosition}px` } as React.CSSProperties}>
      <Layout.Vertical>
        <Container className={css.diffHeader} height={DIFF_VIEWER_HEADER_HEIGHT}>
          <Layout.Horizontal>
            <Button
              variation={ButtonVariation.ICON}
              icon={collapsed ? 'main-chevron-right' : 'main-chevron-down'}
              size={ButtonSize.SMALL}
              onClick={() => setCollapsed(!collapsed)}
              iconProps={{
                size: 10,
                style: {
                  color: '#383946',
                  flexGrow: 1,
                  justifyContent: 'center',
                  display: 'flex'
                }
              }}
              className={css.chevron}
            />
            <Text inline className={css.fname}>
              <Link
                to={routes.toCODERepository({
                  repoPath: repoMetadata.path as string,
                  gitRef: pullRequestMetadata?.source_branch,
                  resourcePath: diff.isRename ? diff.newName : diff.filePath
                })}>
                {diff.isRename ? `${diff.oldName} -> ${diff.newName}` : diff.filePath}
              </Link>
              <CopyButton content={diff.filePath} icon={CodeIcon.Copy} size={ButtonSize.SMALL} />
            </Text>
            <Container style={{ alignSelf: 'center' }} padding={{ left: 'small' }}>
              <Layout.Horizontal spacing="xsmall">
                <Render when={diff.addedLines || diff.isNew}>
                  <Text tag="span" className={css.addedLines}>
                    +{diff.addedLines || 0}
                  </Text>
                </Render>
                <Render when={diff.deletedLines || diff.isDeleted}>
                  <Text tag="span" className={css.deletedLines}>
                    -{diff.deletedLines || 0}
                  </Text>
                </Render>
              </Layout.Horizontal>
            </Container>
            <FlexExpander />

            <Render
              when={
                !readOnly &&
                commitRange?.length === 0 &&
                diff.fileViews?.get(diff.filePath) !== undefined &&
                diff.fileViews?.get(diff.filePath) !== diff.checksumAfter
              }>
              <Container>
                <Text className={css.fileChanged}>{getString('changedSinceLastView')}</Text>
              </Container>
            </Render>

            <Render when={!readOnly && commitRange?.length === 0}>
              <Container>
                <label className={css.viewLabel}>
                  <Checkbox
                    checked={viewed}
                    onChange={async () => {
                      if (viewed) {
                        setViewed(false)

                        // update local data first
                        diff.fileViews?.delete(diff.filePath)

                        // best effort attempt to recflect on server (swallow exception - user still sees correct data locally)
                        await unmarkViewed(null, { pathParams: { filePath: diff.filePath } }).catch(() => undefined)
                      } else {
                        setViewed(true)

                        // update local data first
                        // we could wait for server response for the guaranteed correct SHA, but this is non-crucial data so it's okay
                        diff.fileViews?.set(diff.filePath, diff.checksumAfter || 'unknown')

                        // best effort attempt to recflect on server (swallow exception - user still sees correct data locally)
                        await markViewed(
                          {
                            path: diff.filePath,
                            commit_sha: pullRequestMetadata?.source_sha
                          },
                          {}
                        ).catch(() => undefined)
                      }
                    }}
                  />
                  {getString('viewed')}
                </label>
              </Container>
            </Render>
          </Layout.Horizontal>
        </Container>

        <Container
          id={diff.contentId}
          data-path={diff.filePath}
          className={cx(css.diffContent, {
            [css.standalone]: standalone,
            [css.navV2]: !!document.querySelector('[data-code-nav-version="2"]'),
            [css.sidebarCollapsed]: !isSidebarExpanded
          })}
          ref={contentRef}>
          <Render when={renderCustomContent}>
            <Container>
              <Layout.Vertical padding="xlarge" style={{ alignItems: 'center' }}>
                <Render when={fileDeleted}>
                  <Button
                    variation={ButtonVariation.LINK}
                    onClick={() => {
                      setRenderCustomContent(false)
                      setTimeout(() => renderDiffAndUpdateContainerHeightIfNeeded(true), 0)
                    }}>
                    {getString('pr.showDiff')}
                  </Button>
                </Render>
                <Text>{getString(fileDeleted ? 'pr.fileDeleted' : 'pr.fileUnchanged')}</Text>
              </Layout.Vertical>
            </Container>
          </Render>
        </Container>
      </Layout.Vertical>
      <NavigationCheck when={dirty} />
    </Container>
  )
}

const SIDE_NAV_EXPANDED_EVENT = 'SIDE_NAV_EXPANDED_EVENT'
