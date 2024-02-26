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
import { noop } from 'lodash-es'
import { useMutate } from 'restful-react'
import { useInView } from 'react-intersection-observer'
import {
  Button,
  Container,
  FlexExpander,
  ButtonVariation,
  Layout,
  Text,
  ButtonSize,
  Checkbox,
  useIsMounted
} from '@harnessio/uicore'
import cx from 'classnames'
import { Render } from 'react-jsx-match'
import { Link } from 'react-router-dom'
import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import type { DiffFileEntry } from 'utils/types'
import { useAppContext } from 'AppContext'
import type { TypesPullReq } from 'services/code'
import { isInViewport } from 'utils/Utils'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import type { UseGetPullRequestInfoResult } from 'pages/PullRequest/useGetPullRequestInfo'
import { dispatchCustomEvent, useCustomEventListener } from 'hooks/useEventListener'
import { useQueryParams } from 'hooks/useQueryParams'
import {
  DIFF2HTML_CONFIG,
  DIFF_VIEWER_HEADER_HEIGHT,
  ViewStyle,
  getFileViewedState,
  FileViewedState
} from './DiffViewerUtils'
import { usePullReqComments } from './usePullReqComments'
import css from './DiffViewer.module.scss'

interface DiffViewerProps extends Pick<GitInfoProps, 'repoMetadata'> {
  diffs: DiffFileEntry[]
  diff: DiffFileEntry
  viewStyle: ViewStyle
  stickyTopPosition?: number
  readOnly?: boolean
  pullRequestMetadata?: TypesPullReq
  targetRef?: string
  sourceRef?: string
  commitRange?: string[]
  scrollElement: HTMLElement
  commitSHA?: string
  refetchActivities?: UseGetPullRequestInfoResult['refetchActivities']
}

//
// Notes:
//       (1) Lots of direct DOM manipulations are used to boost performance.
//       Try to avoid React re-rendering at all cost.
//
//       (2) This component focuses very much on rendering a diff. Handling
//       PR comments are consolidated in usePullReqComments hook.
//
//       (3) DOM is expensive. The more DOM we have on a page, the slower it
//       is - does not matter if they are rendered by native JS or React.
//       The less DOMs we have, the better performance is gained.
//
//       (4) For active DOMs that we can't destroy, minimize their impact on
//       performance by hidding them (visibility: hidden), or using the new
//       CSS property `content-visibility`.
//
const DiffViewerInternal: React.FC<DiffViewerProps> = ({
  diffs,
  diff,
  viewStyle,
  stickyTopPosition = 0,
  readOnly,
  repoMetadata,
  pullRequestMetadata: pullReqMetadata,
  targetRef,
  sourceRef,
  commitRange,
  scrollElement,
  commitSHA,
  refetchActivities
}) => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const internalFlags = useRef({ isContentEmpty: true })
  const viewedPath = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata?.number}/file-views`,
    [repoMetadata.path, pullReqMetadata?.number]
  )
  const { mutate: markViewed } = useMutate({ verb: 'PUT', path: viewedPath })
  const { mutate: unmarkViewed } = useMutate({ verb: 'DELETE', path: ({ filePath }) => `${viewedPath}/${filePath}` })

  const { path } = useQueryParams<{ path: string; commentId: string }>()
  const shouldDiffBeShownByDefault = useMemo(() => path === diff.filePath, [path, diff.filePath])
  const diffHasVeryLongLine = useMemo(
    () => diff.blocks?.some(block => block.lines?.some(line => line.content?.length > LONG_LINE_SIZE_LIMIT)),
    [diff]
  )

  // File viewed feature is only enabled if no commit range is provided (otherwise component is hidden, too)
  const [viewed, setViewed] = useState(
    commitRange?.length === 0 &&
      getFileViewedState(diff.filePath, diff.checksumAfter, diff.fileViews) === FileViewedState.VIEWED &&
      !shouldDiffBeShownByDefault
  )

  // When `path` is specified in url, the diff associated to this path has highest priority to
  // be rendered first. This diff can be at anywhere the list so the chance for it to lock to
  // itself is near zero - as others might have higher change to render themselves. To avoid
  // others to render before the lock is enforced, any DiffViewer instance, that sees `path` and
  // finds the lock is still available, will set this lock (even might not for itself) anyway.
  useEffect(function setLockToSingleDiffAnyway() {
    if (path && !lockSingleDiffRenderingAtFilePath) {
      lockSingleDiffRenderingAtFilePath = path
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (commitRange?.length === 0) {
      setViewed(getFileViewedState(diff.filePath, diff.checksumAfter, diff.fileViews) === FileViewedState.VIEWED)
    }
  }, [setViewed, diff.fileViews, diff.filePath, diff.checksumAfter, commitRange])

  const showChangedSinceLastView = useMemo(
    () =>
      !readOnly &&
      commitRange?.length === 0 &&
      getFileViewedState(diff.filePath, diff.checksumAfter, diff.fileViews) === FileViewedState.CHANGED,
    [readOnly, commitRange?.length, diff.filePath, diff.checksumAfter, diff.fileViews]
  )
  const [collapsed, setCollapsed] = useState(viewed)
  const isBinary = useMemo(() => diff.isBinary, [diff.isBinary])
  const fileUnchanged = useMemo(() => diff.unchangedPercentage === 100, [diff.unchangedPercentage])
  const fileDeleted = useMemo(() => diff.isDeleted, [diff.isDeleted])
  const isDiffTooLarge = useMemo(
    () => diff.addedLines + diff.deletedLines > DIFF_CHANGES_LIMIT,
    [diff.addedLines, diff.deletedLines]
  )
  const [renderCustomContent, setRenderCustomContent] = useState(
    !shouldDiffBeShownByDefault && (fileUnchanged || fileDeleted || isDiffTooLarge || isBinary || diffHasVeryLongLine)
  )
  const { ref, inView } = useInView({
    rootMargin: `${IN_VIEW_MARGIN}px ${IN_VIEW_MARGIN}px`,
    //
    // This flag is important to make sure handleDiffAndCommentsVisibility is not
    // called twice initially for 1st file (one with inView false, one with true).
    // Note: Do not set `delay`, as event could be handled incorrectly when page
    // is scrolled too quickly.
    //
    initialInView: true
  })

  const containerRef = useRef<HTMLDivElement | null>(null)
  const setContainerRef = useCallback(
    node => {
      containerRef.current = node
      ref(node)
    },
    [ref]
  )

  const contentRef = useRef<HTMLDivElement>(null)
  const diff2HtmlRef = useRef<{ renderer: Diff2HtmlUI; diff: DiffFileEntry }>()
  const diffHandlerRafRef = useRef(0)
  const isMounted = useIsMounted()
  const [dirty, setDirty] = useState(false)

  //
  // Handling custom events sent to DiffViewer from external components/features
  // such as "jump to file", "jump to comment", etc...
  // Note: This useCustomEventListener() hook must be called before usePullReqComments()
  // hook in order for usePullReqComments to send back events properly.
  //
  useCustomEventListener<DiffViewerCustomEvent>(
    diff.filePath,
    useCallback(event => {
      const { action, onDone, index, commentId } = event.detail
      const containerDOM = document.getElementById(diff.containerId) as HTMLDivElement

      function scrollToContainer() {
        if (!isMounted.current) return

        containerDOM.scrollIntoView({ block: 'start' })

        if (!commentId) {
          // Check to adjust scroll position to make sure content is not
          // cut off due to current scroll position
          const scrollGap = containerDOM.getBoundingClientRect().y - stickyTopPosition
          if (scrollGap < 1) {
            scrollElement.scroll({ top: (scrollElement.scrollTop || window.scrollY) + scrollGap })
          }
        } else {
          const commentDOM = containerDOM.querySelector(`[data-comment-id="${commentId}"]`) as HTMLDivElement
          // dom is the great grand parent of the comment DOM (CommentBox)
          const dom = commentDOM?.parentElement?.parentElement?.parentElement?.parentElement
          if (dom) dom.lastElementChild?.scrollIntoView({ block: 'center' })
        }
      }

      function dispatchRenderIfInViewEventToSibling(diffIndex: number, distance: number) {
        const siblingIndex = diffIndex + distance

        if (!isMounted.current || siblingIndex < 0 || siblingIndex > diffs.length - 1) return

        // Always scroll back to the current diff container or its comment, otherwise, when
        // siblings are rendered, scroll position is moved with them
        scrollToContainer()

        dispatchCustomEvent<DiffViewerCustomEvent>(diffs[siblingIndex].filePath, {
          action: DiffViewerEvent.RENDER_IF_INVIEW,
          diffs,
          index,
          onDone: () => dispatchRenderIfInViewEventToSibling(siblingIndex, distance)
        })
      }

      switch (action) {
        case DiffViewerEvent.SCROLL_INTO_VIEW: {
          // When scrolling too quickly over diffs (jump to a file), the diff rendering
          //  pipeline is super expensive and costly due to many diffs receive the inView
          // event. To optimize, use a flag to render a single diff (the one we scroll into)
          // and skip rendering of all others. When this diff rendering is done, dispatch
          // RENDER_IF_INVIEW event to the diff siblings to consider rendering themselves.
          // This approach is much more practical than rendering every single diffs that
          // being scrolled by.
          lockSingleDiffRenderingAtFilePath = diff.filePath
          scrollToContainer()

          handleDiffAndCommentsVisibility(true, function onRendered() {
            setTimeout(() => {
              if (!isMounted.current) return

              scrollToContainer()
              lockSingleDiffRenderingAtFilePath = ''

              // Ask (above, below) siblings to consider render themselves
              dispatchRenderIfInViewEventToSibling(index, -1)
              dispatchRenderIfInViewEventToSibling(index, +1)
            }, 50)
          })
          break
        }

        case DiffViewerEvent.RENDER_IF_INVIEW: {
          handleDiffAndCommentsVisibility(false, onDone)
          break
        }
      }
    }, []), // eslint-disable-line react-hooks/exhaustive-deps
    () => !!diff.filePath
  )

  const commentsHook = usePullReqComments({
    diffs,
    diff,
    viewStyle,
    stickyTopPosition,
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
  })

  useEffect(
    function alwaysExpandDiffIfChangedSinceLastView() {
      if (showChangedSinceLastView && collapsed && !viewed) {
        setCollapsed(false)
      }
    },
    [showChangedSinceLastView, viewed] // eslint-disable-line react-hooks/exhaustive-deps
  )

  //
  // Offload diff contents. The purpose of this function:
  //  - Detach all comment threads, if they exist
  //  - Set container height to a fixed number to avoid flickering
  //  - Destroy diff contents to reduce number of DOM elements on page
  //  - Set container visibility to hidden if requested so
  //
  const offloadDiffContents = useCallback(
    (setContainerVisibilityHidden: boolean) => {
      if (!isMounted.current) return

      const containerDOM = containerRef.current as HTMLDivElement
      const contentDOM = contentRef.current as HTMLDivElement
      const { style } = containerDOM

      // Detach all comment threads
      if (!readOnly) commentsHook.current.detachAllCommentThreads()

      style.height = AUTO
      if (setContainerVisibilityHidden) style.visibility = HIDDEN
      style.height = containerDOM.clientHeight + 'px'
      contentDOM.innerText = ''
      internalFlags.current.isContentEmpty = true
    },
    [isMounted, commentsHook, readOnly]
  )

  //
  // Handle diff content and comments rendering and visibility.
  //
  const handleDiffAndCommentsVisibility = useCallback(
    (ignoreChecks = false, onRendered = noop) => {
      const containerDOM = containerRef.current as HTMLDivElement
      const contentDOM = contentRef.current as HTMLDivElement
      const { style: containerStyle } = containerDOM
      const { style: contentStyle } = contentDOM
      const { visibility } = containerStyle
      const { isContentEmpty } = internalFlags.current
      const isContainerInViewport = inView || isInViewport(containerDOM, IN_VIEW_MARGIN)

      if (renderCustomContent) {
        onRendered()
        return
      }

      // Single diff rendering is locked
      if (lockSingleDiffRenderingAtFilePath) {
        if (lockSingleDiffRenderingAtFilePath !== diff.filePath) {
          // Not for this diff, return
          return
        } else if (diffHandlerRafRef.current > 0 && !ignoreChecks) {
          // Lock is for this diff
          // Check if there's a task already schedule, and if this call has
          // low priority (!ignoreChecks), then return
          return
        }
      }

      cancelAnimationFrame(diffHandlerRafRef.current)

      if (collapsed) {
        if (!isContentEmpty) offloadDiffContents(false)
      } else {
        if (isContainerInViewport && (visibility !== VISIBLE || isContentEmpty || !collapsed || ignoreChecks)) {
          // Schedule a task to decide if we should render diff
          // content (and comments). RAF helps:
          //  (1)- Reduce unresponsive UI
          //  (2)- Avoid rendering if container is already out of viewport at the
          //       time RAF callback is executed
          diffHandlerRafRef.current = requestAnimationFrame(() => {
            if (!isMounted.current) return

            // Need to check if container in viewport again when RAF callback
            // is executed. At this time, the container may be out of viewport
            // already. ONLY render diff (and comments) when container is in
            // viewport.
            if (isInViewport(containerDOM, IN_VIEW_MARGIN) || visibility === HIDDEN) {
              if (diff2HtmlRef.current?.diff !== diff) {
                diff2HtmlRef.current = {
                  renderer: new Diff2HtmlUI(
                    contentDOM,
                    [diff],
                    Object.assign({}, DIFF2HTML_CONFIG, { outputFormat: viewStyle })
                  ),
                  diff
                }
              }

              // Set container height to auto, allowing new comments to be added
              // inside diff content without having to adjusting its height manually
              containerStyle.height = AUTO

              // Since attachAllCommentThreads renders comment threads in separate
              // React roots. Give React time to render the comments completely
              // before showing the whole container. Otherwise, there'd be a
              // flicker (diff rendered first, then comments rendered a while
              // later due to React scheduling)
              if (!visibility) contentStyle.visibility = HIDDEN

              // Draw diff content and attach all comment threads
              diff2HtmlRef.current.renderer.draw()

              if (!readOnly) {
                commentsHook.current.attachAllCommentThreads()
              }

              // Use a flag to save content empty state (avoid contentDOM.innerText
              // access and comparision which is costly)
              internalFlags.current.isContentEmpty = false

              // Give React some time to process attachAllCommentThreads
              if (readOnly) {
                containerStyle.visibility = VISIBLE
                contentStyle.visibility = VISIBLE
                onRendered()
              } else {
                setTimeout(() => {
                  if (isMounted.current) {
                    containerStyle.visibility = VISIBLE
                    contentStyle.visibility = VISIBLE
                  }
                  onRendered()
                }, 100)
              }

              // Reset RAF id when task is done
              diffHandlerRafRef.current = 0
            }
          })
        } else if (!isContainerInViewport && visibility === VISIBLE) {
          // Schedule a task to offload diff contents, and hide container. This is key to gain
          // performance. When a diff is out of viewport, destroy diff HTML contents so browser
          // can get back allocated memory, plus it has less DOMs to worry about. Secondly, set
          // the container visibility to hidden helps reduce browser load in terms of event
          // handling against the DOMs.
          diffHandlerRafRef.current = requestAnimationFrame(() => {
            offloadDiffContents(true)
            // Reset RAF id when task is done
            diffHandlerRafRef.current = 0
          })
        }
      }

      return () => cancelAnimationFrame(diffHandlerRafRef.current)
    },
    [readOnly, commentsHook, inView, collapsed, diff, viewStyle, isMounted, offloadDiffContents, renderCustomContent]
  )

  useEffect(handleDiffAndCommentsVisibility, [handleDiffAndCommentsVisibility])

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
            <Text
              inline
              className={css.fname}
              lineClamp={1}
              tooltipProps={{
                portalClassName: css.popover,
                className: css.fnamePopover
              }}>
              <Link
                to={routes.toCODERepository({
                  repoPath: repoMetadata.path as string,
                  gitRef: pullReqMetadata?.source_branch || commitSHA || '',
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

            <Render when={showChangedSinceLastView}>
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
                        setCollapsed(false)

                        // update local data first
                        diff.fileViews?.delete(diff.filePath)

                        // best effort attempt to recflect on server (swallow exception - user still sees correct data locally)
                        await unmarkViewed(null, { pathParams: { filePath: diff.filePath } }).catch(noop)
                      } else {
                        setViewed(true)
                        setCollapsed(true)

                        // update local data first
                        // we could wait for server response for the guaranteed correct SHA, but this is non-crucial data so it's okay
                        diff.fileViews?.set(diff.filePath, diff.checksumAfter || 'unknown')

                        // best effort attempt to recflect on server (swallow exception - user still sees correct data locally)
                        await markViewed(
                          {
                            path: diff.filePath,
                            commit_sha: pullReqMetadata?.source_sha
                          },
                          {}
                        ).catch(noop)
                      }
                    }}
                  />
                  {getString('viewed')}
                </label>
              </Container>
            </Render>
          </Layout.Horizontal>
        </Container>

        <Container id={diff.contentId} data-path={diff.filePath} className={css.diffContent} ref={contentRef}>
          <Render when={renderCustomContent && !collapsed}>
            <Container height={200} flex={{ align: 'center-center' }}>
              <Layout.Vertical padding="xlarge" style={{ alignItems: 'center' }}>
                <Render when={fileDeleted || isDiffTooLarge || diffHasVeryLongLine}>
                  <Button
                    variation={ButtonVariation.LINK}
                    onClick={() => {
                      setRenderCustomContent(false)
                      handleDiffAndCommentsVisibility(true)
                    }}>
                    {getString('pr.showDiff')}
                  </Button>
                </Render>
                <Text>
                  {getString(
                    fileDeleted
                      ? 'pr.fileDeleted'
                      : isDiffTooLarge || diffHasVeryLongLine
                      ? 'pr.diffTooLarge'
                      : isBinary
                      ? 'pr.fileBinary'
                      : 'pr.fileUnchanged'
                  )}
                </Text>
              </Layout.Vertical>
            </Container>
          </Render>
        </Container>
      </Layout.Vertical>
      <NavigationCheck when={dirty} />
    </Container>
  )
}

let lockSingleDiffRenderingAtFilePath = ''

const IN_VIEW_MARGIN = 1200
const AUTO = 'auto'
const VISIBLE = 'visible'
const HIDDEN = 'hidden'

// If diff has a line that exceeds LONG_LINE_SIZE_LIMIT characters, consider
// it being a large diff
const LONG_LINE_SIZE_LIMIT = 5000

// addedLines + deletedLines > DIFF_CHANGES_LIMIT ? "Large diffs are not rendered by default"
const DIFF_CHANGES_LIMIT = 1000

export enum DiffViewerEvent {
  SCROLL_INTO_VIEW = 'scrollIntoView',
  RENDER_IF_INVIEW = 'renderIfInView'
}

export interface DiffViewerCustomEvent {
  action: DiffViewerEvent
  diffs: DiffFileEntry[]
  index: number
  commentId?: string
  onDone: () => void
}

export const DiffViewer = React.memo(DiffViewerInternal)
