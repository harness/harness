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

import React from 'react'
import { Avatar, Container, Layout, StringSubstitute, Text } from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Render } from 'react-jsx-match'
import { defaultTo } from 'lodash-es'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { useStrings } from 'framework/strings'
import type { TypesPullReqActivity } from 'services/code'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import { formatDate, formatTime, PullRequestSection } from 'utils/Utils'
import { CommentType } from 'components/DiffViewer/DiffViewerUtils'
import { useAppContext } from 'AppContext'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import css from './Conversation.module.scss'

interface SystemCommentProps extends Pick<GitInfoProps, 'pullReqMetadata'> {
  commentItems: CommentItem<TypesPullReqActivity>[]
  repoMetadataPath?: string
}

export const SystemComment: React.FC<SystemCommentProps> = ({ pullReqMetadata, commentItems, repoMetadataPath }) => {
  const { getString } = useStrings()
  const payload = commentItems[0].payload
  const type = payload?.type
  const { routes } = useAppContext()

  switch (type) {
    case CommentType.MERGE: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Container margin={{ left: 'xsmall' }} width={24} height={24} className={css.mergeContainer}>
              <Icon name={CodeIcon.Merged} size={16} color={Color.PURPLE_700} />
            </Container>

            <Avatar name={pullReqMetadata.merger?.display_name} size="small" hoverCard={false} />
            <Text>
              <StringSubstitute
                str={getString('pr.prMergedInfo')}
                vars={{
                  user: <strong>{pullReqMetadata.merger?.display_name}</strong>,
                  source: <strong>{pullReqMetadata.source_branch}</strong>,
                  target: <strong>{pullReqMetadata.target_branch}</strong>,
                  time: (
                    <Text inline margin={{ left: 'xsmall' }}>
                      <PipeSeparator height={9} />
                      <TimePopoverWithLocal
                        time={defaultTo(pullReqMetadata.merged as number, 0)}
                        inline={false}
                        className={css.timeText}
                        font={{ variation: FontVariation.SMALL }}
                        color={Color.GREY_400}
                      />
                    </Text>
                  )
                }}
              />
            </Text>
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.REVIEW_SUBMIT: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Icon
              margin={{ left: 'small' }}
              padding={{ right: 'small' }}
              {...generateReviewDecisionIcon((payload?.payload as Unknown)?.decision)}
            />

            <Avatar name={payload?.author?.display_name as string} size="small" hoverCard={false} />
            <Text color={Color.GREY_500}>
              <StringSubstitute
                str={getString('pr.prReviewSubmit')}
                vars={{
                  user: <strong>{payload?.author?.display_name}</strong>,
                  state: <Text margin={{ right: 'xsmall' }}>{(payload?.payload as Unknown)?.decision}</Text>,
                  time: (
                    <Text inline margin={{ left: 'xsmall' }}>
                      <PipeSeparator height={9} />
                      <TimePopoverWithLocal
                        time={defaultTo(payload?.created as number, 0)}
                        inline={false}
                        className={css.timeText}
                        font={{ variation: FontVariation.SMALL }}
                        color={Color.GREY_400}
                      />
                    </Text>
                  )
                }}
              />
            </Text>
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.BRANCH_UPDATE: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text flex tag="div">
              <StringSubstitute
                str={getString('pr.prBranchPushInfo')}
                vars={{
                  user: (
                    <Text padding={{ right: 'small' }} inline>
                      <strong>{payload?.author?.display_name}</strong>
                    </Text>
                  ),
                  commit: (
                    <Container className={css.commitContainer} padding={{ left: 'small' }}>
                      <CommitActions
                        enableCopy
                        sha={(payload?.payload as Unknown)?.new}
                        href={routes.toCODEPullRequest({
                          repoPath: repoMetadataPath as string,
                          pullRequestSection: PullRequestSection.FILES_CHANGED,
                          pullRequestId: String(pullReqMetadata.number),
                          commitSHA: (payload?.payload as Unknown)?.new as string
                        })}
                      />
                    </Container>
                  )
                }}
              />
            </Text>
            <PipeSeparator height={9} />
            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.BRANCH_DELETE: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text flex tag="div">
              <StringSubstitute
                str={getString('pr.prBranchDeleteInfo')}
                vars={{
                  user: (
                    <Text padding={{ right: 'small' }} inline>
                      <strong>{payload?.author?.display_name}</strong>
                    </Text>
                  ),
                  commit: (
                    <Container className={css.commitContainer} padding={{ left: 'small' }}>
                      <CommitActions
                        enableCopy
                        sha={(payload?.payload as Unknown)?.sha}
                        href={routes.toCODEPullRequest({
                          repoPath: repoMetadataPath as string,
                          pullRequestSection: PullRequestSection.FILES_CHANGED,
                          pullRequestId: String(pullReqMetadata.number),
                          commitSHA: (payload?.payload as Unknown)?.sha as string
                        })}
                      />
                    </Container>
                  )
                }}
              />
            </Text>
            <PipeSeparator height={9} />
            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              width={100}
              inline={true}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.STATE_CHANGE: {
      const openFromDraft =
        (payload?.payload as Unknown)?.old_draft === true && (payload?.payload as Unknown)?.new_draft === false

      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text>
              <StringSubstitute
                str={getString(openFromDraft ? 'pr.prStateChangedDraft' : 'pr.prStateChanged')}
                vars={{
                  user: <strong>{payload?.author?.display_name}</strong>,
                  old: <strong>{(payload?.payload as Unknown)?.old}</strong>,
                  new: <strong>{(payload?.payload as Unknown)?.new}</strong>
                }}
              />
            </Text>
            <PipeSeparator height={9} />
            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.TITLE_CHANGE: {
      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <StringSubstitute
                str={getString('pr.titleChanged')}
                vars={{
                  user: <strong>{payload?.author?.display_name}</strong>,
                  old: (
                    <strong>
                      <s>{(payload?.payload as Unknown)?.old}</s>
                    </strong>
                  ),
                  new: <strong>{(payload?.payload as Unknown)?.new}</strong>
                }}
              />
            </Text>
            <PipeSeparator height={9} />

            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
          <Render when={commentItems.length > 1}>
            <Container
              margin={{ top: 'medium', left: 'xxxlarge' }}
              style={{ maxWidth: 'calc(100vw - 450px)', overflow: 'auto' }}>
              <MarkdownViewer
                source={[getString('pr.titleChangedTable').replace(/\n$/, '')]
                  .concat(
                    commentItems
                      .filter((_, index) => index > 0)
                      .map(
                        item =>
                          `|${item.author}|<s>${(item.payload?.payload as Unknown)?.old}</s>|${
                            (item.payload?.payload as Unknown)?.new
                          }|${formatDate(item.edited)} ${formatTime(item.edited)}|`
                      )
                  )
                  .join('\n')}
              />
            </Container>
          </Render>
        </Container>
      )
    }

    default: {
      // eslint-disable-next-line no-console
      console.warn('Unable to render system type activity', commentItems)
      return (
        <Text className={css.mergedBox}>
          <Icon name={CodeIcon.Commit} padding={{ right: 'small' }} />
          {type}
        </Text>
      )
    }
  }
}

const generateReviewDecisionIcon = (
  reviewDecision: string
): {
  name: IconName
  color: string | undefined
  size: number | undefined
  icon: IconName
  iconProps?: { color?: Color }
} => {
  let icon: IconName = 'dot'
  let color: Color | undefined = undefined
  let size: number | undefined = undefined

  switch (reviewDecision) {
    case 'changereq':
      icon = 'main-issue-filled'
      color = Color.ORANGE_700
      size = 18
      break
    case 'approved':
      icon = 'execution-success'
      size = 18
      color = Color.GREEN_700
      break
  }
  const name = icon
  return { name, color, size, icon, ...(color ? { iconProps: { color } } : undefined) }
}
