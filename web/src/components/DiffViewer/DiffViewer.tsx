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
import { useGet, useMutate } from 'restful-react'
import {
  Button,
  Container,
  FlexExpander,
  ButtonVariation,
  Layout,
  Text,
  ButtonSize,
  Checkbox,
  useIsMounted,
  useToaster
} from '@harnessio/uicore'
import cx from 'classnames'
import * as Diff2Html from 'diff2html'
import { Render } from 'react-jsx-match'
import { Link } from 'react-router-dom'
import { useInView } from 'react-intersection-observer'
import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import type { DiffFileEntry } from 'utils/types'
import { useAppContext } from 'AppContext'
import type { GitFileDiff, TypesPullReq } from 'services/code'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import type { UseGetPullRequestInfoResult } from 'pages/PullRequest/useGetPullRequestInfo'
import { useQueryParams } from 'hooks/useQueryParams'
import { useCustomEventListener } from 'hooks/useEventListener'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { getErrorMessage, isInViewport } from 'utils/Utils'
import { createRequestIdleCallbackTaskPool } from 'utils/Task'
import { useResizeObserver } from 'hooks/useResizeObserver'
import { useFindGitBranch } from 'hooks/useFindGitBranch'
import Config from 'Config'
import {
  DIFF2HTML_CONFIG,
  DIFF_VIEWER_HEADER_HEIGHT,
  ViewStyle,
  getFileViewedState,
  FileViewedState
} from './DiffViewerUtils'
import { usePullReqComments } from './usePullReqComments'
import Collapse from '../../icons/collapse.svg'
import Expand from '../../icons/expand.svg'
import css from './DiffViewer.module.scss'

interface DiffViewerProps extends Pick<GitInfoProps, 'repoMetadata'> {
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
  memorizedState: Map<string, DiffViewerExchangeState>
  fullDiffAPIPath: string
}

const DiffViewerInternal: React.FC<DiffViewerProps> = ({
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
  refetchActivities,
  memorizedState,
  fullDiffAPIPath
}) => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { showError } = useToaster()
  const viewedPath = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata?.number}/file-views`,
    [repoMetadata.path, pullReqMetadata?.number]
  )
  const { mutate: markViewed } = useMutate({ verb: 'PUT', path: viewedPath })
  const { mutate: unmarkViewed } = useMutate({ verb: 'DELETE', path: ({ filePath }) => `${viewedPath}/${filePath}` })

  const { path } = useQueryParams<{ path: string; commentId: string }>()
  const shouldDiffBeShownByDefault = useMemo(() => path === diff.filePath, [path, diff.filePath])
  const diffHasVeryLongLine = useMemo(
    () => diff.blocks?.some(block => block.lines?.some(line => line.content?.length > Config.MAX_TEXT_LINE_SIZE_LIMIT)),
    [diff]
  )

  // File viewed feature is only enabled if no commit range is provided (otherwise component is hidden, too)
  const [viewed, setViewed] = useState(
    commitRange?.length === 0 &&
      getFileViewedState(diff.filePath, diff.checksumAfter, diff.fileViews) === FileViewedState.VIEWED &&
      !shouldDiffBeShownByDefault
  )

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
  const [collapsed, setCollapsed] = useState(viewed || !!memorizedState.get(diff.filePath)?.collapsed)
  const isBinary = useMemo(() => diff.isBinary, [diff.isBinary])
  const fileUnchanged = useMemo(
    () => diff.unchangedPercentage === 100 || (diff.addedLines === 0 && diff.deletedLines === 0),
    [diff.addedLines, diff.deletedLines, diff.unchangedPercentage]
  )
  const fileDeleted = useMemo(() => diff.isDeleted, [diff.isDeleted])
  const isDiffTooLarge = useMemo(
    () => diff.addedLines + diff.deletedLines > Config.PULL_REQUEST_LARGE_DIFF_CHANGES_LIMIT,
    [diff.addedLines, diff.deletedLines]
  )
  const [renderCustomContent, setRenderCustomContent] = useState(
    !shouldDiffBeShownByDefault && (fileUnchanged || fileDeleted || isDiffTooLarge || isBinary || diffHasVeryLongLine)
  )
  const containerRef = useRef<HTMLDivElement | null>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const diff2HtmlRef = useRef<{ renderer: Diff2HtmlUI; diff: DiffFileEntry }>()
  const [dirty, setDirty] = useState(false)
  const isMounted = useIsMounted()
  const [useFullDiff, setUseFullDiff] = useState(!!memorizedState.get(diff.filePath)?.useFullDiff)
  const { ref, inView } = useInView({
    rootMargin: `500px 0px 500px 0px`,
    initialInView: true
  })
  const setContainerRef = useCallback(
    node => {
      containerRef.current = node
      ref(node)
    },
    [ref]
  )

  useResizeObserver(
    contentRef,
    useCallback(
      dom => {
        if (isMounted.current && dom) {
          dom.style.setProperty(BLOCK_HEIGHT, dom.clientHeight + 'px')
        }
      },
      [isMounted]
    )
  )

  useEffect(() => {
    let taskId = 0
    if (inView) {
      taskId = scheduleLowPriorityTask(() => {
        if (isMounted.current && contentRef.current) contentRef.current.classList.remove(css.hidden)
      })
    } else {
      taskId = scheduleLowPriorityTask(() => {
        if (isMounted.current && contentRef.current) contentRef.current.classList.add(css.hidden)
      })
    }
    return () => cancelTask(taskId)
  }, [inView, isMounted])

  //
  // Handling custom events sent to DiffViewer from external components/features
  // such as "jump to file", "jump to comment", etc...
  //
  useCustomEventListener<DiffViewerCustomEvent>(
    diff.filePath,
    useCallback(event => {
      const { action, commentId } = event.detail
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

      switch (action) {
        case DiffViewerEvent.SCROLL_INTO_VIEW: {
          scrollToContainer()
          break
        }
      }
    }, []), // eslint-disable-line react-hooks/exhaustive-deps
    () => !!diff.filePath
  )

  const commentsHook = usePullReqComments({
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

  const renderDiffAndComments = useCallback(() => {
    if (!isMounted.current) return

    const fullDiff = memorizedState.get(diff.filePath)?.fullDiff
    const _diff = useFullDiff && fullDiff ? fullDiff : diff

    // Create a new diff renderer if cached diff is different from current diff
    // to ensure when new commit is selected, the diff is re-rendered correctly
    if (diff2HtmlRef.current?.diff !== _diff) {
      diff2HtmlRef.current = {
        renderer: new Diff2HtmlUI(
          contentRef.current as HTMLDivElement,
          [_diff],
          Object.assign({}, DIFF2HTML_CONFIG, { outputFormat: viewStyle })
        ),
        diff: _diff
      }
    }

    diff2HtmlRef.current.renderer.draw()
    commentsHook.current.attachAllCommentThreads()
  }, [commentsHook, diff, memorizedState, useFullDiff, viewStyle, isMounted])

  useEffect(
    function renderDiffAndCommentsIfInViewportOrSchedule() {
      let taskId = 0

      if (!renderCustomContent && !collapsed) {
        if (isInViewport(containerRef.current as Element, 1000)) {
          renderDiffAndComments()
        } else {
          taskId = scheduleLowPriorityTask(renderDiffAndComments)
        }
      }

      memorizedState.set(diff.filePath, { ...memorizedState.get(diff.filePath), collapsed })

      return () => cancelTask(taskId)
    },
    [collapsed, diff.filePath, memorizedState, isMounted, renderDiffAndComments, renderCustomContent]
  )

  const {
    data: fullDiffData,
    error: fullDiffError,
    loading: fullDiffLoading,
    refetch: getFullDiff,
    cancel: cancelGetFullDiff
  } = useGet<GitFileDiff[]>({
    path: fullDiffAPIPath,
    requestOptions: { headers: { Accept: 'application/json' } },
    queryParams: { include_patch: true, path: diff.filePath, range: 1 },
    lazy: !useFullDiff || !!memorizedState.get(diff.filePath)?.fullDiff
  })

  const branchInfo = useFindGitBranch(pullReqMetadata?.source_branch)

  useShowRequestError(fullDiffError, 0)

  useEffect(
    function parseAndAssignFullDiff() {
      if (fullDiffData) {
        try {
          memorizedState.set(diff.filePath, {
            ...memorizedState.get(diff.filePath),
            fullDiff: Diff2Html.parse(
              window.atob((fullDiffData[0].patch as unknown as string) || ''),
              DIFF2HTML_CONFIG
            ).map(_diff => ({ ...diff, ..._diff }))[0],
            useFullDiff: true
          })

          setUseFullDiff(true)
          setRenderCustomContent(false)

          if (memorizedState.get(diff.filePath)?.collapsed) {
            setCollapsed(false)
            memorizedState.set(diff.filePath, { ...memorizedState.get(diff.filePath), collapsed: false })
          }
        } catch (exception) {
          showError(getErrorMessage(exception), 0)
        }
      }
    },
    [diff, diff.filePath, memorizedState, fullDiffData, showError]
  )

  useEffect(
    function adjustScrollPositionWhenCollapsingFile() {
      const containerDOM = containerRef.current as HTMLDivElement

      if (
        containerDOM &&
        !useFullDiff &&
        memorizedState.get(diff.filePath)?.useFullDiff === false &&
        !isInViewport(containerDOM)
      ) {
        if (stickyTopPosition && containerDOM.getBoundingClientRect().y - stickyTopPosition < 1) {
          containerDOM.scrollIntoView()
          scrollElement.scroll({ top: (scrollElement.scrollTop || window.scrollY) - stickyTopPosition })
        }
      }
    },
    [scrollElement, stickyTopPosition, useFullDiff, diff, memorizedState]
  )

  const toggleFullDiff = useCallback(() => {
    // If full diff is not fetched, fetch it and set useFullDiff when data arrives
    // Otherwise, toggle useFullDiff flag
    if (!memorizedState.get(diff.filePath)?.fullDiff && !fullDiffLoading) {
      cancelGetFullDiff()
      getFullDiff()
    } else {
      memorizedState.set(diff.filePath, { ...memorizedState.get(diff.filePath), useFullDiff: !useFullDiff })
      setUseFullDiff(!useFullDiff)
    }
  }, [useFullDiff, memorizedState, diff.filePath, cancelGetFullDiff, getFullDiff, fullDiffLoading])

  const ToggleFullDiffIcon = useMemo(() => (useFullDiff ? Collapse : Expand), [useFullDiff])

  return (
    <Container
      ref={setContainerRef}
      id={diff.containerId}
      className={cx(css.main, { [css.readOnly]: readOnly })}
      data-diff-file-path={diff.filePath}
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
                size: 12,
                style: {
                  color: '#383946',
                  flexGrow: 1,
                  justifyContent: 'center',
                  display: 'flex'
                }
              }}
              className={css.chevron}
            />
            <Button
              variation={ButtonVariation.ICON}
              className={css.expandCollapseDiffBtn}
              onClick={toggleFullDiff}
              title={getString(useFullDiff ? 'pr.collapseFullFile' : 'pr.expandFullFile')}>
              <ToggleFullDiffIcon width="16" height="16" strokeWidth="2" />
            </Button>
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
                  gitRef: pullReqMetadata?.source_branch
                    ? branchInfo
                      ? pullReqMetadata?.source_branch
                      : pullReqMetadata?.source_sha
                    : commitSHA || '',
                  resourcePath: diff.isRename ? diff.newName : diff.filePath
                })}>
                {diff.isRename ? `${diff.oldName} -> ${diff.newName}` : diff.filePath}
              </Link>
              <CopyButton content={diff.filePath} icon={CodeIcon.Copy} size={ButtonSize.SMALL} />
            </Text>
            <Container style={{ alignSelf: 'center' }} padding={{ left: 'small' }} margin={{ right: 'small' }}>
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

            <Render when={!readOnly}>
              <Container>
                <Layout.Horizontal spacing="xsmall" flex>
                  {fullDiffLoading && (
                    <Icon name="steps-spinner" color={Color.PRIMARY_7} margin={{ right: 'xsmall' }} />
                  )}
                  <Render when={commitRange?.length === 0}>
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
                  </Render>
                </Layout.Horizontal>
              </Container>
            </Render>
          </Layout.Horizontal>
        </Container>

        <Container id={diff.contentId} data-path={diff.filePath} className={css.diffContent} ref={contentRef}>
          <Render when={renderCustomContent && !collapsed}>
            <Container height={200} flex={{ align: 'center-center' }}>
              <Layout.Vertical padding="xlarge" style={{ alignItems: 'center' }}>
                <Render when={fileDeleted || isDiffTooLarge || diffHasVeryLongLine}>
                  <Button variation={ButtonVariation.LINK} onClick={() => setRenderCustomContent(false)}>
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

const BLOCK_HEIGHT = '--block-height'

export enum DiffViewerEvent {
  SCROLL_INTO_VIEW = 'scrollIntoView'
}

export interface DiffViewerCustomEvent {
  action: DiffViewerEvent
  commentId?: string
}

export interface DiffViewerExchangeState {
  collapsed?: boolean
  useFullDiff?: boolean
  fullDiff?: DiffFileEntry
}

const { scheduleTask: scheduleLowPriorityTask, cancelTask } = createRequestIdleCallbackTaskPool()

export const DiffViewer = React.memo(DiffViewerInternal)
