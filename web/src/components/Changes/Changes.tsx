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

import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  Container,
  FlexExpander,
  ButtonVariation,
  Layout,
  Text,
  StringSubstitute,
  Button,
  PageError,
  useIsMounted
} from '@harnessio/uicore'
import { atom, useAtom } from 'jotai'
import { Match, Case, Render } from 'react-jsx-match'
import * as Diff2Html from 'diff2html'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { isEqual, noop, throttle } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { normalizeGitRef, type GitInfoProps, FILE_VIEWED_OBSOLETE_SHA } from 'utils/GitUtils'
import { formatNumber, getErrorMessage, PullRequestSection, voidFn } from 'utils/Utils'
import { DiffViewer } from 'components/DiffViewer/DiffViewer'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import type { DiffFileEntry } from 'utils/types'
import { DIFF2HTML_CONFIG, ViewStyle } from 'components/DiffViewer/DiffViewerUtils'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import type { TypesPullReqFileView, TypesPullReq, TypesListCommitResponse } from 'services/code'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { useAppContext } from 'AppContext'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { PlainButton } from 'components/PlainButton/PlainButton'
import { useEventListener } from 'hooks/useEventListener'
import type { UseGetPullRequestInfoResult } from 'pages/PullRequest/useGetPullRequestInfo'
import { ChangesDropdown } from './ChangesDropdown'
import { DiffViewConfiguration } from './DiffViewConfiguration'
import ReviewSplitButton from './ReviewSplitButton/ReviewSplitButton'
import CommitRangeDropdown from './CommitRangeDropdown/CommitRangeDropdown'
import css from './Changes.module.scss'

const STICKY_TOP_POSITION = 64
const STICKY_HEADER_HEIGHT = 150
const changedFileId = (collection: Unknown[]) => collection.filter(Boolean).join('::::')

interface ChangesProps extends Pick<GitInfoProps, 'repoMetadata'> {
  targetRef?: string
  sourceRef?: string
  commitSHA?: string
  readOnly?: boolean
  emptyTitle: string
  emptyMessage: string
  pullRequestMetadata?: TypesPullReq
  pullReqCommits?: TypesListCommitResponse
  className?: string
  defaultCommitRange?: string[]
  scrollElement: HTMLElement
  refetchActivities?: UseGetPullRequestInfoResult['refetchActivities']
  refetchCommits?: UseGetPullRequestInfoResult['refetchCommits']
}

const ChangesInternal: React.FC<ChangesProps> = ({
  repoMetadata,
  targetRef: _targetRef,
  sourceRef: _sourceRef,
  commitSHA,
  readOnly,
  emptyTitle,
  emptyMessage,
  pullRequestMetadata,
  pullReqCommits,
  className,
  defaultCommitRange = [],
  scrollElement,
  refetchActivities,
  refetchCommits
}) => {
  const { getString } = useStrings()
  const [viewStyle, setViewStyle] = useUserPreference(UserPreference.DIFF_VIEW_STYLE, ViewStyle.SIDE_BY_SIDE)
  const [lineBreaks, setLineBreaks] = useUserPreference(UserPreference.DIFF_LINE_BREAKS, false)
  const [diffs, setDiffs] = useState<DiffFileEntry[]>([])
  const headerRef = useRef<HTMLDivElement | null>(null)
  const scrollTopRef = useRef<HTMLDivElement | null>(null)
  const [commitRange, setCommitRange] = useState(defaultCommitRange)
  const [prHasChanged, setPrHasChanged] = useState(false)
  const [sourceRef, setSourceRef] = useState(_sourceRef)
  const [targetRef, setTargetRef] = useState(_targetRef)
  const isMounted = useIsMounted()

  const diffApiPath = useMemo(
    () =>
      commitSHA
        ? // show specific commit
          `commits/${commitSHA}/diff`
        : // show range of commits and user selected subrange
        commitRange.length > 0
        ? `diff/${commitRange[0]}~1...${commitRange[commitRange.length - 1]}`
        : // show range of commits and user did not select a subrange
          `diff/${normalizeGitRef(targetRef)}...${normalizeGitRef(sourceRef)}`,
    [commitSHA, commitRange, targetRef, sourceRef]
  )
  const path = useMemo(() => `/api/v1/repos/${repoMetadata?.path}/+/${diffApiPath}`, [repoMetadata?.path, diffApiPath])
  const [cachedDiff, setCachedDiff] = useAtom(changesInfoAtom)

  const {
    data: rawDiff,
    error,
    loading: loadingRawDiff,
    refetch
  } = useGet<string>({
    path,
    requestOptions: { headers: { Accept: 'text/plain' } },
    lazy: cachedDiff.path === path ? true : commitSHA ? false : !targetRef || !sourceRef
  })

  // In readOnly mode (Compare page), we'd like to refetch diff immediately when source
  // or target refs changed from Compare page. Otherwise (PullRequest page), we'll need
  // to confirm with user if they want to refresh (as they might be reviewing the PR)
  useEffect(
    function updateInternalRefsOnReadOnlyMode() {
      if (readOnly && (_sourceRef !== sourceRef || _targetRef !== targetRef)) {
        setSourceRef(ref => (ref !== _sourceRef ? _sourceRef : ref))
        setTargetRef(ref => (ref !== _targetRef ? _targetRef : ref))
      }
    },
    [readOnly, _sourceRef, _targetRef, sourceRef, targetRef]
  )

  const {
    data: fileViewsData,
    loading: loadingFileViews,
    error: errorFileViews,
    refetch: refetchFileViews
  } = useGet<TypesPullReqFileView[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/file-views`,
    lazy: true
  })

  useEffect(() => {
    if (pullRequestMetadata?.number && !fileViewsData && !cachedDiff.fileViews) {
      refetchFileViews()
    }
  }, [pullRequestMetadata?.number, fileViewsData, refetchFileViews, cachedDiff.fileViews])

  const loading = useMemo(
    () => (loadingRawDiff && !cachedDiff.raw) || (loadingFileViews && !cachedDiff.fileViews),
    [loadingRawDiff, loadingFileViews, cachedDiff.fileViews, cachedDiff.raw]
  )
  const diffStats = useMemo(
    () =>
      (diffs || []).reduce(
        (obj, diff) => {
          obj.addedLines += diff.addedLines
          obj.deletedLines += diff.deletedLines
          return obj
        },
        { addedLines: 0, deletedLines: 0 }
      ),
    [diffs]
  )
  const shouldHideReviewButton = useMemo(
    () => readOnly || pullRequestMetadata?.state === 'merged' || pullRequestMetadata?.state === 'closed',
    [readOnly, pullRequestMetadata?.state]
  )
  const { currentUser } = useAppContext()
  const isActiveUserPROwner = useMemo(
    () =>
      !!currentUser?.uid && !!pullRequestMetadata?.author?.uid && currentUser?.uid === pullRequestMetadata?.author?.uid,
    [currentUser, pullRequestMetadata]
  )

  useEffect(() => {
    if (sourceRef !== _sourceRef || targetRef !== _targetRef) {
      setPrHasChanged(true)
    }
  }, [_sourceRef, sourceRef, _targetRef, targetRef])

  useEffect(
    function updateCacheWhenDiffDataArrives() {
      if (path && rawDiff && typeof rawDiff === 'string') {
        const fileViews = readOnly
          ? new Map<string, string>()
          : fileViewsData
              ?.filter(({ path: _path, sha }) => _path && sha)
              .reduce((map, { path: _path, sha, obsolete }) => {
                map.set(_path as string, (obsolete ? FILE_VIEWED_OBSOLETE_SHA : sha) as string)
                return map
              }, new Map<string, string>())

        setCachedDiff({ path, raw: rawDiff, fileViews })
      }
    },
    [rawDiff, path, setCachedDiff, fileViewsData, readOnly]
  )

  //
  // Parsing diff and construct data structure to pass into DiffViewer component
  //
  useEffect(() => {
    if (loadingRawDiff || cachedDiff.path !== path || typeof cachedDiff.raw !== 'string' || !cachedDiff.fileViews) {
      return
    }

    if (cachedDiff.raw) {
      const _diffs = Diff2Html.parse(cachedDiff.raw, DIFF2HTML_CONFIG).map(diff => {
        const fileId = changedFileId([diff.oldName, diff.newName])
        const containerId = `container-${fileId}`
        const contentId = `content-${fileId}`
        const filePath = diff.isDeleted ? diff.oldName : diff.newName

        return {
          ...diff,
          containerId,
          contentId,
          fileId,
          filePath,
          fileViews: cachedDiff.fileViews
        }
      })

      setDiffs(oldDiffs => (isEqual(oldDiffs, _diffs) ? oldDiffs : _diffs))
    } else {
      setDiffs([])
    }
  }, [readOnly, path, cachedDiff, loadingRawDiff])

  //
  // Listen to scroll event to toggle "scroll to top" button
  //
  useEventListener(
    'scroll',
    useMemo(() => {
      let done = false
      let scrollRafId = 0

      return throttle(() => {
        cancelAnimationFrame(scrollRafId)

        scrollRafId = requestAnimationFrame(() => {
          if (isMounted.current && headerRef.current && scrollTopRef.current) {
            const { classList: headerCL } = headerRef.current
            const { classList: scrollTopCL } = scrollTopRef.current
            const isSticky = (scrollElement.scrollTop || window.scrollY) >= STICKY_HEADER_HEIGHT

            if (isSticky) {
              if (!done) {
                headerCL.add(css.stickied)
                scrollTopCL.add(css.show)
                done = true
              }
            } else if (done) {
              headerCL.remove(css.stickied)
              scrollTopCL.remove(css.show)
              done = false
            }
          }
        })
      }, 150)
    }, []), // eslint-disable-line react-hooks/exhaustive-deps
    scrollElement,
    {
      capture: true,
      passive: true
    }
  )

  const history = useHistory()
  const { routes } = useAppContext()
  const commitSHARef = useRef(commitSHA)

  useEffect(
    function updatePageWhenCommitRangeIsChanged() {
      if (!pullRequestMetadata?.number && !isMounted.current) {
        return
      }

      if (commitSHARef.current !== commitSHA) {
        commitSHARef.current = commitSHA
        history.push(
          routes.toCODEPullRequest({
            repoPath: repoMetadata.path as string,
            pullRequestId: String(pullRequestMetadata?.number),
            pullRequestSection: PullRequestSection.FILES_CHANGED,
            commitSHA
          })
        )
      }
    },
    [isMounted, commitRange, history, routes, repoMetadata.path, pullRequestMetadata?.number, commitSHA]
  )

  useShowRequestError(errorFileViews, 0)

  return (
    <Container className={cx(css.container, className)} {...(!!loadingRawDiff || !!error ? { flex: true } : {})}>
      <LoadingSpinner visible={loading} withBorder={true} />
      <Render when={error}>
        <PageError message={getErrorMessage(error)} onClick={voidFn(refetch)} />
      </Render>
      <Render when={!error && !loadingRawDiff}>
        <Container className={css.header} ref={headerRef}>
          <Layout.Horizontal>
            <Container flex={{ alignItems: 'center' }}>
              <CommitRangeDropdown
                allCommits={pullReqCommits?.commits || []}
                selectedCommits={commitRange}
                setSelectedCommits={setCommitRange}
              />

              {/* Files Changed stats */}
              <Text flex className={css.diffStatsLabel}>
                <StringSubstitute
                  str={getString('pr.diffStatsLabel')}
                  vars={{
                    changedFilesLink: <ChangesDropdown diffs={diffs} />,
                    addedLines: diffStats.addedLines ? formatNumber(diffStats.addedLines) : '0',
                    deletedLines: diffStats.deletedLines ? formatNumber(diffStats.deletedLines) : '0',
                    configuration: (
                      <DiffViewConfiguration
                        viewStyle={viewStyle}
                        setViewStyle={setViewStyle}
                        lineBreaks={lineBreaks}
                        setLineBreaks={setLineBreaks}
                      />
                    )
                  }}
                />
              </Text>

              <Render when={prHasChanged && !readOnly && commitRange?.length === 0}>
                <PlainButton
                  text={getString('refresh')}
                  className={css.refreshBtn}
                  onClick={() => {
                    setCachedDiff({})
                    setTargetRef(_targetRef)
                    setSourceRef(_sourceRef)
                    setPrHasChanged(false)
                    refetchCommits?.()
                  }}
                />
              </Render>

              {/* Show "Scroll to top" button */}
              <Layout.Horizontal ref={scrollTopRef} className={css.scrollTop}>
                <PipeSeparator height={10} />
                <Button
                  variation={ButtonVariation.ICON}
                  icon="arrow-up"
                  iconProps={{ size: 14 }}
                  onClick={() => scrollElement.scroll({ top: 0 })}
                  tooltip={getString('scrollToTop')}
                  tooltipProps={{ isDark: true }}
                />
              </Layout.Horizontal>
            </Container>

            <FlexExpander />

            <ReviewSplitButton
              shouldHide={shouldHideReviewButton}
              repoMetadata={repoMetadata}
              pullRequestMetadata={pullRequestMetadata}
              refreshPr={voidFn(noop)}
              disabled={isActiveUserPROwner}
            />
          </Layout.Horizontal>
        </Container>
      </Render>
      <Render when={!loadingRawDiff && !error}>
        <Match expr={diffs?.length}>
          <Case val={(len: number) => len > 0}>
            {/* TODO: lineBreaks is broken in line-by-line view, enable it for side-by-side only now */}
            <Layout.Vertical
              spacing="medium"
              className={cx(css.main, {
                // TODO: Line break barely works. Disable until we find a complete solution for it
                // https://harness.atlassian.net/browse/CODE-1452
                // [css.enableDiffLineBreaks]: lineBreaks && viewStyle === ViewStyle.SIDE_BY_SIDE
              })}>
              {/*
               * TODO: Render diffs by blocks, not everything at once
               */}
              {diffs.map((diff, index) => (
                // Note: `key={viewStyle + index + lineBreaks}` resets DiffView when view configuration
                // is changed. Making it easier to control states inside DiffView itself, as it does not
                //  have to deal with any view configuration
                <DiffViewer
                  key={viewStyle + diffApiPath + index + lineBreaks}
                  readOnly={readOnly || (commitRange?.length || 0) > 0} // render in readonly mode in case a commit is selected
                  diffs={diffs}
                  diff={diff}
                  viewStyle={viewStyle}
                  stickyTopPosition={STICKY_TOP_POSITION}
                  repoMetadata={repoMetadata}
                  pullRequestMetadata={pullRequestMetadata}
                  targetRef={targetRef}
                  sourceRef={_sourceRef}
                  commitRange={commitRange}
                  scrollElement={scrollElement}
                  commitSHA={commitSHA}
                  refetchActivities={refetchActivities}
                />
              ))}
            </Layout.Vertical>
          </Case>
          <Case val={0}>
            <Container padding="xlarge">
              <NoResultCard
                showWhen={() => diffs?.length === 0 && !loadingRawDiff && !loading}
                forSearch={true}
                title={emptyTitle}
                emptySearchMessage={emptyMessage}
              />
            </Container>
          </Case>
        </Match>
      </Render>
    </Container>
  )
}

export const Changes = React.memo(ChangesInternal)

const changesInfoAtom = atom<{ path?: string; raw?: string; fileViews?: Map<string, string> }>({})
