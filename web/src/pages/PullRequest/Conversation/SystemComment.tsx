import React from 'react'
import {
  Avatar,
  Color,
  Container,
  FontVariation,
  Icon,
  IconName,
  Layout,
  StringSubstitute,
  Text
} from '@harness/uicore'
import ReactTimeago from 'react-timeago'
import { Render } from 'react-jsx-match'
import { Link } from 'react-router-dom'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { useStrings } from 'framework/strings'
import type { TypesPullReqActivity } from 'services/code'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import { formatDate, formatTime } from 'utils/Utils'
import { CommentType } from 'components/DiffViewer/DiffViewerUtils'
import { useAppContext } from 'AppContext'
import css from './Conversation.module.scss'

interface SystemCommentProps extends Pick<GitInfoProps, 'pullRequestMetadata'> {
  commentItems: CommentItem<TypesPullReqActivity>[]
  repoMetadataPath?: string
}

export const SystemComment: React.FC<SystemCommentProps> = ({
  pullRequestMetadata,
  commentItems,
  repoMetadataPath
}) => {
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

            <Avatar name={pullRequestMetadata.merger?.display_name} size="small" hoverCard={false} />
            <Text>
              <StringSubstitute
                str={getString('pr.prMergedInfo')}
                vars={{
                  user: <strong>{pullRequestMetadata.merger?.display_name}</strong>,
                  source: <strong>{pullRequestMetadata.source_branch}</strong>,
                  target: <strong>{pullRequestMetadata.target_branch} </strong>,
                  time: <ReactTimeago date={pullRequestMetadata.merged as number} />
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
                  state: (payload?.payload as Unknown)?.decision,
                  time: <ReactTimeago className={css.timeText} date={payload?.created as number} />
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
            <Text>
              <StringSubstitute
                str={getString('pr.prBranchPushInfo')}
                vars={{
                  user: <strong>{payload?.author?.display_name}</strong>,
                  commit: (
                    <Link
                      className={css.commitLink}
                      to={routes.toCODECommit({
                        repoPath: repoMetadataPath as string,
                        commitRef: (payload?.payload as Unknown)?.new
                      })}>
                      {(payload?.payload as Unknown)?.new.substring(0, 6)}
                    </Link>
                  )
                }}
              />
            </Text>
            <Text
              inline
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
              width={100}
              style={{ textAlign: 'right' }}>
              <ReactTimeago date={payload?.created as number} />
            </Text>
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

            <Text
              inline
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
              width={100}
              style={{ textAlign: 'right' }}>
              <ReactTimeago date={payload?.created as number} />
            </Text>
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

            <Text
              inline
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
              width={100}
              style={{ textAlign: 'right' }}>
              <ReactTimeago date={payload?.created as number} />
            </Text>
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
                          }|${formatDate(item.updated)} ${formatTime(item.updated)}|`
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
