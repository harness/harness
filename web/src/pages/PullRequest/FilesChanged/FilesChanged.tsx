import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Container, FlexExpander, ButtonVariation, Layout, Text, StringSubstitute, Button } from '@harness/uicore'
import { noop } from 'lodash-es'
import * as Diff2Html from 'diff2html'
// import { useAppContext } from 'AppContext'
// import { useStrings } from 'framework/strings'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import { formatNumber } from 'utils/Utils'
import { DiffViewer } from 'components/DiffViewer/DiffViewer'
import { useEventListener } from 'hooks/useEventListener'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import type { DiffFileEntry } from 'utils/types'
import { DIFF2HTML_CONFIG, ViewStyle } from 'components/DiffViewer/DiffViewerUtils'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import { FilesChangedDropdown } from './FilesChangedDropdown'
import { DiffViewConfiguration } from './DiffViewConfiguration'
import css from './FilesChanged.module.scss'
import diffExample from 'raw-loader!./example.diff'

const STICKY_TOP_POSITION = 64
const STICKY_HEADER_HEIGHT = 150
const diffViewerId = (collection: Unknown[]) => collection.filter(Boolean).join('::::')

export const FilesChanged: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = () => {
  const { getString } = useStrings()
  const [viewStyle, setViewStyle] = useUserPreference(UserPreference.DIFF_VIEW_STYLE, ViewStyle.SIDE_BY_SIDE)
  const [diffs, setDiffs] = useState<DiffFileEntry[]>([])
  const [isSticky, setSticky] = useState(false)
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

  useEffect(() => {
    setDiffs(
      Diff2Html.parse(diffExample, DIFF2HTML_CONFIG).map(diff => {
        const viewerId = diffViewerId([diff.oldName, diff.newName])
        const containerId = `container-${viewerId}`
        const contentId = `content-${viewerId}`

        return {
          ...diff,
          containerId,
          contentId
        }
      })
    )
  }, [])

  useEventListener(
    'scroll',
    useCallback(() => setSticky(window.scrollY >= STICKY_HEADER_HEIGHT), [])
  )

  return (
    <PullRequestTabContentWrapper loading={undefined} error={undefined} onRetry={noop} className={css.wrapper}>
      <Container className={css.header}>
        <Layout.Horizontal>
          <Container flex={{ alignItems: 'center' }}>
            {/* Files Changed stats */}
            <Text icon="accordion-collapsed" iconProps={{ size: 12 }} className={css.diffStatsLabel}>
              <StringSubstitute
                str={getString('pr.diffStatsLabel')}
                vars={{
                  changedFilesLink: <FilesChangedDropdown diffs={diffs} />,
                  addedLines: formatNumber(diffStats.addedLines),
                  deletedLines: formatNumber(diffStats.deletedLines),
                  configuration: <DiffViewConfiguration viewStyle={viewStyle} setViewStyle={setViewStyle} />
                }}
              />
            </Text>

            {/* Show "Scroll to top" button */}
            {isSticky && (
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
            )}
          </Container>
          <FlexExpander />
          <Button text={getString('pr.reviewChanges')} variation={ButtonVariation.PRIMARY} intent="success" />
        </Layout.Horizontal>
      </Container>

      <Layout.Vertical spacing="medium" className={css.diffs}>
        {diffs?.map((diff, index) => (
          // Note: `key={viewStyle + index}` will reset DiffView when viewStyle
          // is changed. Making it easier to control states inside DiffView itself, as it does not have to deal with viewStyle
          <DiffViewer
            key={viewStyle + index}
            diff={diff}
            viewStyle={viewStyle}
            stickyTopPosition={STICKY_TOP_POSITION}
          />
        ))}
      </Layout.Vertical>
    </PullRequestTabContentWrapper>
  )
}
