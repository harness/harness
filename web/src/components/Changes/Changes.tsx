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
} from '@harness/uicore'
import { Match, Case, Render } from 'react-jsx-match'
import * as Diff2Html from 'diff2html'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { noop } from 'lodash-es'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import { formatNumber, getErrorMessage, voidFn } from 'utils/Utils'
import { DiffViewer } from 'components/DiffViewer/DiffViewer'
import { useEventListener } from 'hooks/useEventListener'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import type { DiffFileEntry } from 'utils/types'
import { DIFF2HTML_CONFIG, ViewStyle } from 'components/DiffViewer/DiffViewerUtils'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import type { TypesPullReq, TypesPullReqActivity } from 'services/code'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { useAppContext } from 'AppContext'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { ChangesDropdown } from './ChangesDropdown'
import { DiffViewConfiguration } from './DiffViewConfiguration'
import ReviewSplitButton from './ReviewSplitButton/ReviewSplitButton'
import css from './Changes.module.scss'

const STICKY_TOP_POSITION = 64
const STICKY_HEADER_HEIGHT = 150
const changedFileId = (collection: Unknown[]) => collection.filter(Boolean).join('::::')

interface ChangesProps extends Pick<GitInfoProps, 'repoMetadata'> {
  targetBranch?: string
  sourceBranch?: string
  readOnly?: boolean
  emptyTitle: string
  emptyMessage: string
  pullRequestMetadata?: TypesPullReq
  className?: string
  onCommentUpdate: () => void
  prHasChanged?: boolean
  onDataReady?: (data: DiffFileEntry[]) => void
}

export const Changes: React.FC<ChangesProps> = ({
  repoMetadata,
  targetBranch,
  sourceBranch,
  readOnly,
  emptyTitle,
  emptyMessage,
  pullRequestMetadata,
  onCommentUpdate,
  className,
  prHasChanged,
  onDataReady
}) => {
  const { getString } = useStrings()
  const [viewStyle, setViewStyle] = useUserPreference(UserPreference.DIFF_VIEW_STYLE, ViewStyle.SIDE_BY_SIDE)
  const [lineBreaks, setLineBreaks] = useUserPreference(UserPreference.DIFF_LINE_BREAKS, false)
  const [diffs, setDiffs] = useState<DiffFileEntry[]>([])
  const [isSticky, setSticky] = useState(false)

  const {
    data: rawDiff,
    error,
    loading,
    refetch,
    response
  } = useGet<string>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/compare/${
      pullRequestMetadata ? `${pullRequestMetadata.merge_base_sha}...${pullRequestMetadata.source_sha}` : `${targetBranch}...${sourceBranch}`
    }`,
    lazy: !targetBranch || !sourceBranch
  })
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
    () => loading || (loadingActivities && !activities),
    [loading, loadingActivities, activities]
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
      if (prActivities) {
        setActivities(prActivities)
      }
    },
    [prActivities]
  )

  useEffect(() => {
    if (prHasChanged) {
      refetchActivities()
    }
  }, [prHasChanged, refetchActivities])

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
          activities: activities || []
        }
      })

      setDiffs(_diffs)
      onDataReady?.(_diffs)
    }
  }, [rawDiff, activities, onDataReady])

  useEventListener(
    'scroll',
    useCallback(() => setSticky(window.scrollY >= STICKY_HEADER_HEIGHT), [])
  )

  useShowRequestError(errorActivities)

  return (
    <Container className={cx(css.container, className)} {...(!!loading || !!error ? { flex: true } : {})}>
      <LoadingSpinner visible={loading || showSpinner} withBorder={true} />
      <Render when={error}>
        <PageError message={getErrorMessage(error || errorActivities)} onClick={voidFn(refetch)} />
      </Render>
      <Render when={!loading && !error}>
        <Match expr={diffs?.length}>
          <Case val={(len: number) => len > 0}>
            <>
              <Container className={cx(css.header, { [css.stickied]: isSticky })}>
                <Layout.Horizontal>
                  <Container flex={{ alignItems: 'center' }}>
                    {/* Files Changed stats */}
                    <Text flex className={css.diffStatsLabel}>
                      <StringSubstitute
                        str={getString('pr.diffStatsLabel')}
                        vars={{
                          changedFilesLink: <ChangesDropdown diffs={diffs} />,
                          addedLines: formatNumber(diffStats.addedLines),
                          deletedLines: formatNumber(diffStats.deletedLines),
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

                    {/* Show "Scroll to top" button */}
                    <Render when={isSticky}>
                      <Layout.Horizontal padding={{ left: 'small' }}>
                        <PipeSeparator height={10} />
                        <Button
                          variation={ButtonVariation.ICON}
                          icon="arrow-up"
                          iconProps={{ size: 14 }}
                          onClick={() => window.scroll({ top: 0 })}
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

              {/* TODO: lineBreaks is broken in line-by-line view, enable it for side-by-side only now */}
              <Layout.Vertical
                spacing="large"
                className={cx(css.main, {
                  [css.enableDiffLineBreaks]: lineBreaks && viewStyle === ViewStyle.SIDE_BY_SIDE
                })}>
                {diffs?.map((diff, index) => (
                  // Note: `key={viewStyle + index + lineBreaks}` resets DiffView when view configuration
                  // is changed. Making it easier to control states inside DiffView itself, as it does not
                  //  have to deal with any view configuration
                  <DiffViewer
                    readOnly={readOnly}
                    key={viewStyle + index + lineBreaks}
                    diff={diff}
                    viewStyle={viewStyle}
                    stickyTopPosition={STICKY_TOP_POSITION}
                    repoMetadata={repoMetadata}
                    pullRequestMetadata={pullRequestMetadata}
                    onCommentUpdate={onCommentUpdate}
                    mergeBaseSHA={response?.headers?.get('x-merge-base-sha') || ''}
                    sourceSHA={response?.headers?.get('x-source-sha') || ''}
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
