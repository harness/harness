import React, { useEffect, useMemo, useState } from 'react'
import {
  Container,
  FlexExpander,
  ButtonVariation,
  Layout,
  Text,
  StringSubstitute,
  FontVariation,
  Button,
  Icon,
  Color
} from '@harness/uicore'
import { ButtonGroup, Button as BButton, Classes, Menu, MenuItem } from '@blueprintjs/core'
import { noop } from 'lodash-es'
import cx from 'classnames'
import * as Diff2Html from 'diff2html'
// import { useAppContext } from 'AppContext'
// import { useStrings } from 'framework/strings'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import type { DiffFile } from 'diff2html/lib/types'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { ButtonRoleProps, formatNumber, waitUntil } from 'utils/Utils'
import { DiffViewer, DIFF2HTML_CONFIG, DiffViewStyle } from 'components/DiffViewer/DiffViewer'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import diffExample from 'raw-loader!./example2.diff'
import css from './PullRequestDiff.module.scss'

const STICKY_TOP_POSITION = 64

export const PullRequestDiff: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = () => {
  const { getString } = useStrings()
  const [viewStyle, setViewStyle] = useUserPreference(UserPreference.DIFF_VIEW_STYLE, DiffViewStyle.SPLIT)
  const [diffs, setDiffs] = useState<DiffFile[]>([])
  const [stickyInAction, setStickyInAction] = useState(false)
  const diffStats = useMemo(() => {
    return (diffs || []).reduce(
      (obj, diff) => {
        obj.addedLines += diff.addedLines
        obj.deletedLines += diff.deletedLines
        return obj
      },
      { addedLines: 0, deletedLines: 0 }
    )
  }, [diffs])

  useEffect(() => {
    setDiffs(Diff2Html.parse(diffExample, DIFF2HTML_CONFIG))
  }, [])

  useEffect(() => {
    const onScroll = () => {
      if (window.scrollY >= 150) {
        if (!stickyInAction) {
          setStickyInAction(true)
        }
      } else {
        if (stickyInAction) {
          setStickyInAction(false)
        }
      }
    }
    window.addEventListener('scroll', onScroll)

    return () => {
      window.removeEventListener('scroll', onScroll)
    }
  }, [stickyInAction])

  // console.log({ diffs, viewStyle })

  return (
    <PullRequestTabContentWrapper loading={undefined} error={undefined} onRetry={noop} className={css.wrapper}>
      <Container className={css.header}>
        <Layout.Horizontal>
          <Container flex={{ alignItems: 'center' }}>
            <Text icon="accordion-collapsed" iconProps={{ size: 12 }} className={css.showLabel}>
              <StringSubstitute
                str={getString('pr.showLabel')}
                vars={{
                  showLink: (
                    <Button
                      variation={ButtonVariation.LINK}
                      className={css.showLabelLink}
                      tooltip={
                        <Menu className={css.filesMenu}>
                          {diffs?.map((diff, index) => (
                            <MenuItem
                              key={index}
                              className={css.menuItem}
                              icon={<Icon name={CodeIcon.File} padding={{ right: 'xsmall' }} />}
                              labelElement={
                                <Layout.Horizontal spacing="xsmall">
                                  {!!diff.addedLines && (
                                    <Text color={Color.GREEN_600} style={{ fontSize: '12px' }}>
                                      +{diff.addedLines}
                                    </Text>
                                  )}
                                  {!!diff.addedLines && !!diff.deletedLines && <PipeSeparator height={8} />}
                                  {!!diff.deletedLines && (
                                    <Text color={Color.RED_500} style={{ fontSize: '12px' }}>
                                      -{diff.deletedLines}
                                    </Text>
                                  )}
                                </Layout.Horizontal>
                              }
                              text={
                                diff.isDeleted
                                  ? diff.oldName
                                  : diff.isRename
                                  ? `${diff.oldName} -> ${diff.newName}`
                                  : diff.newName
                              }
                              onClick={() => {
                                const containerDOM = document.getElementById(`file-diff-container-${index}`)

                                if (containerDOM) {
                                  containerDOM.scrollIntoView()
                                  waitUntil(
                                    () => !!containerDOM.querySelector('[data-rendered="true"]'),
                                    () => {
                                      containerDOM.scrollIntoView()
                                      // Fix scrolling position messes up with sticky header
                                      const { y } = containerDOM.getBoundingClientRect()
                                      if (y - STICKY_TOP_POSITION < 1) {
                                        if (STICKY_TOP_POSITION) {
                                          window.scroll({ top: window.scrollY - STICKY_TOP_POSITION })
                                        }
                                      }
                                    }
                                  )
                                }
                              }}
                            />
                          ))}
                        </Menu>
                      }
                      tooltipProps={{ interactionKind: 'click', hasBackdrop: true }}>
                      <StringSubstitute str={getString('pr.showLink')} vars={{ count: diffs?.length || 0 }} />
                    </Button>
                  ),
                  addedLines: formatNumber(diffStats.addedLines),
                  deletedLines: formatNumber(diffStats.deletedLines),
                  config: (
                    <Text
                      icon="cog"
                      rightIcon="caret-down"
                      tooltip={
                        <Container padding="large">
                          <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                            <Text width={100} font={{ variation: FontVariation.SMALL_BOLD }}>
                              {getString('pr.diffView')}
                            </Text>
                            <ButtonGroup>
                              <BButton
                                className={cx(
                                  Classes.POPOVER_DISMISS,
                                  viewStyle === DiffViewStyle.SPLIT ? Classes.ACTIVE : ''
                                )}
                                onClick={() => {
                                  setViewStyle(DiffViewStyle.SPLIT)
                                  window.scroll({ top: 0 })
                                }}>
                                {getString('pr.split')}
                              </BButton>
                              <BButton
                                className={cx(
                                  Classes.POPOVER_DISMISS,
                                  viewStyle === DiffViewStyle.UNIFIED ? Classes.ACTIVE : ''
                                )}
                                onClick={() => {
                                  setViewStyle(DiffViewStyle.UNIFIED)
                                  window.scroll({ top: 0 })
                                }}>
                                {getString('pr.unified')}
                              </BButton>
                            </ButtonGroup>
                          </Layout.Horizontal>
                        </Container>
                      }
                      iconProps={{ size: 14, padding: { right: 3 } }}
                      rightIconProps={{ size: 13, padding: { left: 0 } }}
                      padding={{ left: 'small' }}
                      {...ButtonRoleProps}
                    />
                  )
                }}
              />
            </Text>
            {stickyInAction && (
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
      <Layout.Vertical spacing="medium" className={css.layout}>
        {diffs?.map((diff, index) => (
          // Note: `key={viewStyle + index}` will reset DiffView when viewStyle
          // is changed. Making it easier to control states inside DiffView itself, as it does not have to deal with viewStyle
          <DiffViewer
            key={viewStyle + index}
            index={index}
            diff={diff}
            viewStyle={viewStyle}
            stickyTopPosition={STICKY_TOP_POSITION}
          />
        ))}
      </Layout.Vertical>
    </PullRequestTabContentWrapper>
  )
}
