import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Container,
  FlexExpander,
  ButtonVariation,
  Layout,
  Text,
  StringSubstitute,
  Button,
  PageError
} from '@harnessio/uicore'
import { Match, Case, Render } from 'react-jsx-match'
import * as Diff2Html from 'diff2html'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { isEqual, noop } from 'lodash-es'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import { PullRequestSection, formatNumber, getErrorMessage, voidFn } from 'utils/Utils'
import { DiffViewer } from 'components/DiffViewer/DiffViewer'
import { useEventListener } from 'hooks/useEventListener'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import type { DiffFileEntry } from 'utils/types'
import { DIFF2HTML_CONFIG, ViewStyle } from 'components/DiffViewer/DiffViewerUtils'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import type { TypesCommit, TypesPullReqFileView, TypesPullReq, TypesPullReqActivity } from 'services/code'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { useAppContext } from 'AppContext'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { PlainButton } from 'components/PlainButton/PlainButton'
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
  readOnly?: boolean
  emptyTitle: string
  emptyMessage: string
  pullRequestMetadata?: TypesPullReq
  className?: string
  onCommentUpdate: () => void
  prStatsChanged: number
  onDataReady?: (data: DiffFileEntry[]) => void
  defaultCommitRange?: string[]
  scrollElement: HTMLElement
}

export const Changes: React.FC<ChangesProps> = ({
  repoMetadata,
  targetRef: _targetRef,
  sourceRef: _sourceRef,
  readOnly,
  emptyTitle,
  emptyMessage,
  pullRequestMetadata,
  className,
  onCommentUpdate,
  prStatsChanged,
  onDataReady,
  defaultCommitRange,
  scrollElement
}) => {
  const { getString } = useStrings()
  const history = useHistory()
  const [viewStyle, setViewStyle] = useUserPreference(UserPreference.DIFF_VIEW_STYLE, ViewStyle.SIDE_BY_SIDE)
  const [lineBreaks, setLineBreaks] = useUserPreference(UserPreference.DIFF_LINE_BREAKS, false)
  const [diffs, setDiffs] = useState<DiffFileEntry[]>([])
  const [isSticky, setSticky] = useState(false)
  const [commitRange, setCommitRange] = useState<string[]>(defaultCommitRange || [])
  const { routes } = useAppContext()
  const [prHasChanged, setPrHasChanged] = useState(false)
  const [sourceRef, setSourceRef] = useState(_sourceRef)
  const [targetRef, setTargetRef] = useState(_targetRef)
  const { data: prCommits } = useGet<{
    commits: TypesCommit[]
  }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      git_ref: sourceRef,
      after: targetRef
    },
    lazy: !pullRequestMetadata?.number
  })

  const commitRangePath = useMemo(
    () =>
      commitRange.length === 1
        ? `${commitRange[0]}~1...${commitRange[0]}`
        : commitRange.length > 1
        ? `${commitRange[0]}~1...${commitRange[commitRange.length - 1]}`
        : undefined,
    [commitRange]
  )

  useEffect(() => {
    if (!pullRequestMetadata) {
      return
    }

    history.push(
      routes.toCODEPullRequest({
        repoPath: repoMetadata.path as string,
        pullRequestId: String(pullRequestMetadata?.number),
        pullRequestSection: PullRequestSection.FILES_CHANGED,
        commitSHA:
          commitRange.length === 0
            ? undefined
            : commitRange.length === 1
            ? commitRange[0]
            : `${commitRange[0]}~1...${commitRange[commitRange.length - 1]}`
      })
    )
  }, [commitRange, history, routes, repoMetadata.path, pullRequestMetadata?.number])

  const {
    data: rawDiff,
    error,
    loading,
    refetch
  } = useGet<string>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/diff/${
      commitRangePath ? commitRangePath : `${targetRef}...${sourceRef}`
    }`,
    requestOptions: {
      headers: {
        Accept: 'text/plain'
      }
    },
    lazy: !targetRef || !sourceRef
  })

  const {
    data: rawFileViews,
    loading: loadingFileViews,
    error: errorFileViews,
    refetch: refetchFileViews
  } = useGet<TypesPullReqFileView[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/file-views`,
    lazy: !pullRequestMetadata?.number
  })

  // create a map for faster lookup and ability to insert / remove single elements
  const fileViews = useMemo(() => {
    const out = new Map<string, string>()
    rawFileViews
      ?.filter(({ path, sha }) => path && sha) // every entry is expected to have a path and sha - otherwise skip ...
      .forEach(({ path, sha, obsolete }) => {
        out.set(path, obsolete ? 'ffffffffffffffffffffffffffffffffffffffff' : sha)
      })
    return out
  }, [rawFileViews])

  const {
    data: prActivities,
    loading: loadingActivities,
    error: errorActivities,
    refetch: refetchActivities
  } = useGet<TypesPullReqActivity[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/activities`,
    lazy: !pullRequestMetadata?.number
  })
  const [activities, setActivities] = useState<TypesPullReqActivity[]>()
  const showSpinner = useMemo(
    () => loading || (loadingActivities && !activities) || (loadingFileViews && !fileViews),
    [loading, loadingActivities, activities, loadingFileViews, fileViews]
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
  const isActiveUserPROwner = useMemo(() => {
    return (
      !!currentUser?.uid && !!pullRequestMetadata?.author?.uid && currentUser?.uid === pullRequestMetadata?.author?.uid
    )
  }, [currentUser, pullRequestMetadata])

  // Optimization to avoid showing unnecessary loading spinner. The trick is to
  // show only the spinner when the component is mounted and not when refetching
  // happens after some comments are authored.
  useEffect(
    function setActivitiesIfNotSet() {
      if (!prActivities || isEqual(activities, prActivities)) {
        return
      }

      setActivities(prActivities)
    },
    [prActivities]
  )

  useEffect(() => {
    if (readOnly) {
      return
    }

    refetchActivities()
  }, [prStatsChanged])

  useEffect(() => {
    if (readOnly) {
      return
    }

    refetchFileViews()
  }, [prStatsChanged])

  useEffect(() => {
    if (sourceRef !== _sourceRef || targetRef !== _targetRef) {
      setPrHasChanged(true)
    }
  }, [_sourceRef, sourceRef, _targetRef, targetRef])

  useEffect(() => {
    const _raw = rawDiff && typeof rawDiff === 'string' ? rawDiff : ''

    if (rawDiff) {
      const _diffs = Diff2Html.parse(_raw, DIFF2HTML_CONFIG).map(diff => {
        const fileId = changedFileId([diff.oldName, diff.newName])
        const containerId = `container-${fileId}`
        const contentId = `content-${fileId}`
        const filePath = diff.isDeleted ? diff.oldName : diff.newName
        const fileActivities: TypesPullReqActivity[] | undefined = activities?.filter(
          activity => filePath === activity.code_comment?.path
        )

        return {
          ...diff,
          containerId,
          contentId,
          fileId,
          filePath,
          fileActivities: fileActivities || [],
          activities: activities || [],
          fileViews: fileViews || []
        }
      })

      setDiffs(_diffs)
      onDataReady?.(_diffs)
    }
  }, [rawDiff, activities, fileViews, onDataReady])

  useEventListener(
    'scroll',
    useCallback(() => {
      setSticky(scrollElement.scrollTop >= STICKY_HEADER_HEIGHT)
    }, []),
    scrollElement
  )

  useShowRequestError(errorActivities)
  useShowRequestError(errorFileViews)

  return (
    <Container className={cx(css.container, className)} {...(!!loading || !!error ? { flex: true } : {})}>
      <LoadingSpinner visible={loading || showSpinner} withBorder={true} />
      <Render when={error}>
        <PageError message={getErrorMessage(error || errorActivities || errorFileViews)} onClick={voidFn(refetch)} />
      </Render>
      <Render when={!error && !loading}>
        <Container className={cx(css.header, { [css.stickied]: isSticky })}>
          <Layout.Horizontal>
            <Container flex={{ alignItems: 'center' }}>
              <Render when={pullRequestMetadata?.number}>
                <CommitRangeDropdown
                  allCommits={prCommits?.commits || []}
                  selectedCommits={commitRange}
                  setSelectedCommits={setCommitRange}
                />
              </Render>

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

              <Render when={prHasChanged}>
                <PlainButton
                  text={getString('refresh')}
                  className={css.refreshBtn}
                  onClick={() => {
                    setPrHasChanged(false)
                    setTargetRef(_targetRef)
                    setSourceRef(_sourceRef)
                  }}
                />
              </Render>

              {/* Show "Scroll to top" button */}
              <Render when={isSticky}>
                <Layout.Horizontal padding={{ left: 'small' }}>
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
              </Render>
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
      <Render when={!loading && !error}>
        <Match expr={diffs?.length}>
          <Case val={(len: number) => len > 0}>
            <>
              {/* TODO: lineBreaks is broken in line-by-line view, enable it for side-by-side only now */}
              <Layout.Vertical
                spacing="medium"
                className={cx(css.main, {
                  [css.enableDiffLineBreaks]: lineBreaks && viewStyle === ViewStyle.SIDE_BY_SIDE
                })}>
                {diffs?.map((diff, index) => (
                  // Note: `key={viewStyle + index + lineBreaks}` resets DiffView when view configuration
                  // is changed. Making it easier to control states inside DiffView itself, as it does not
                  //  have to deal with any view configuration
                  <DiffViewer
                    readOnly={readOnly || (commitRange?.length || 0) > 0} // render in readonly mode in case a commit is selected
                    key={viewStyle + index + lineBreaks}
                    diff={diff}
                    viewStyle={viewStyle}
                    stickyTopPosition={STICKY_TOP_POSITION}
                    repoMetadata={repoMetadata}
                    pullRequestMetadata={pullRequestMetadata}
                    onCommentUpdate={onCommentUpdate}
                    targetRef={targetRef}
                    sourceRef={_sourceRef}
                    commitRange={commitRange}
                    scrollElement={scrollElement}
                  />
                ))}
              </Layout.Vertical>
            </>
          </Case>
          <Case val={0}>
            <Container padding="xlarge">
              <NoResultCard
                showWhen={() => diffs?.length === 0}
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
