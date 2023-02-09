import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Button,
  ButtonSize,
  ButtonVariation,
  Color,
  Container,
  FlexExpander,
  FontVariation,
  Icon,
  Layout,
  StringSubstitute,
  Text,
  useToaster
} from '@harness/uicore'
import cx from 'classnames'
import { useGet, useMutate } from 'restful-react'
import ReactTimeago from 'react-timeago'
import { orderBy } from 'lodash-es'
import { Render } from 'react-jsx-match'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { OpenapiUpdatePullReqRequest, TypesPullReqActivity } from 'services/code'
import { CommentAction, CommentBox, CommentBoxOutletPosition, CommentItem } from 'components/CommentBox/CommentBox'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { formatDate, formatTime, getErrorMessage } from 'utils/Utils'
import {
  activityToCommentItem,
  CommentType,
  PullRequestCodeCommentPayload
} from 'components/DiffViewer/DiffViewerUtils'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import { PullRequestActionsBox } from './PullRequestActionsBox/PullRequestActionsBox'
import PullRequestSideBar from './PullRequestSideBar/PullRequestSideBar'
import css from './Conversation.module.scss'

interface ConversationProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  onCommentUpdate: () => void
  prHasChanged?: boolean
  handleRefresh?: () => void
}

export const Conversation: React.FC<ConversationProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onCommentUpdate,
  prHasChanged,
  handleRefresh
}) => {
  const { getString } = useStrings()
  const { currentUser } = useAppContext()
  const {
    data: activities,
    loading,
    error,
    refetch: refetchActivities
  } = useGet<TypesPullReqActivity[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/activities`
  })
  const { data: reviewers } = useGet<any[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/reviewers`
  })
  const { showError } = useToaster()
  const [newComments, setNewComments] = useState<TypesPullReqActivity[]>([])
  const activityBlocks = useMemo(() => {
    // Each block may have one or more activities which are grouped into it. For example, one comment block
    // contains a parent comment and multiple replied comments
    const blocks: CommentItem<TypesPullReqActivity>[][] = []

    // Determine all parent activities
    const parentActivities = orderBy(activities?.filter(activity => !activity.parent_id) || [], 'edited', 'asc').map(
      _comment => [_comment]
    )

    // Then add their children as follow-up elements (same array)
    parentActivities?.forEach(parentActivity => {
      const childActivities = activities?.filter(activity => activity.parent_id === parentActivity[0].id)

      childActivities?.forEach(childComment => {
        parentActivity.push(childComment)
      })
    })

    parentActivities?.forEach(parentActivity => {
      blocks.push(parentActivity.map(activityToCommentItem))
    })

    if (newComments.length) {
      blocks.push(orderBy(newComments, 'edited', 'asc').map(activityToCommentItem))
    }

    // Group title-change events into one single block
    // Disabled for now, @see https://harness.atlassian.net/browse/SCM-79
    // const titleChangeItems =
    //   blocks.filter(
    //     _activities => isSystemComment(_activities) && _activities[0].payload?.type === CommentType.TITLE_CHANGE
    //   ) || []

    // titleChangeItems.forEach((value, index) => {
    //   if (index > 0) {
    //     titleChangeItems[0].push(...value)
    //   }
    // })
    // titleChangeItems.shift()
    // return blocks.filter(_activities => !titleChangeItems.includes(_activities))

    return blocks
  }, [activities, newComments])
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/comments`,
    [repoMetadata.path, pullRequestMetadata.number]
  )
  const { mutate: saveComment } = useMutate({ verb: 'POST', path })
  const { mutate: updateComment } = useMutate({ verb: 'PATCH', path: ({ id }) => `${path}/${id}` })
  const { mutate: deleteComment } = useMutate({ verb: 'DELETE', path: ({ id }) => `${path}/${id}` })
  const confirmAct = useConfirmAct()
  const [commentCreated, setCommentCreated] = useState(false)
  const refreshPR = useCallback(() => {
    onCommentUpdate()
    refetchActivities()
  }, [onCommentUpdate, refetchActivities])

  useAnimateNewCommentBox(commentCreated, setCommentCreated)

  return (
    <PullRequestTabContentWrapper loading={loading} error={error} onRetry={refetchActivities}>
      <Container>
        <Layout.Vertical spacing="xlarge">
          <PullRequestActionsBox
            repoMetadata={repoMetadata}
            pullRequestMetadata={pullRequestMetadata}
            onPRStateChanged={refreshPR}
          />
          <Container>
            <Layout.Horizontal width={`70%`}>
              <FlexExpander />
              {!prHasChanged ? null : (
                <Button
                  onClick={handleRefresh}
                  iconProps={{ className: css.refreshIcon, size: 12 }}
                  icon="repeat"
                  text={getString('refresh')}
                  variation={ButtonVariation.SECONDARY}
                  size={ButtonSize.SMALL}
                  margin={{ bottom: 'small' }}
                />
              )}
            </Layout.Horizontal>

            <Layout.Horizontal>
              <Container width={`70%`}>
                <Layout.Vertical spacing="xlarge">
                  <DescriptionBox
                    repoMetadata={repoMetadata}
                    pullRequestMetadata={pullRequestMetadata}
                    onCommentUpdate={onCommentUpdate}
                  />

                  {activityBlocks?.map((blocks, index) => {
                    const threadId = blocks[0].payload?.id
                    const commentItems = blocks

                    if (isSystemComment(commentItems)) {
                      return (
                        <SystemBox
                          key={threadId}
                          pullRequestMetadata={pullRequestMetadata}
                          commentItems={commentItems}
                        />
                      )
                    }

                    return (
                      <CommentBox
                        key={threadId}
                        fluid
                        className={cx({
                          [css.newCommentCreated]: commentCreated && index === activityBlocks.length - 1
                        })}
                        getString={getString}
                        commentItems={commentItems}
                        currentUserName={currentUser.display_name}
                        handleAction={async (action, value, commentItem) => {
                          let result = true
                          let updatedItem: CommentItem<TypesPullReqActivity> | undefined = undefined
                          const id = (commentItem as CommentItem<TypesPullReqActivity>)?.payload?.id

                          switch (action) {
                            case CommentAction.DELETE:
                              result = false
                              await confirmAct({
                                message: getString('deleteCommentConfirm'),
                                action: async () => {
                                  await deleteComment({}, { pathParams: { id } })
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

                            case CommentAction.REPLY:
                              await saveComment({ text: value, parent_id: Number(threadId) })
                                .then(newComment => {
                                  updatedItem = activityToCommentItem(newComment)
                                })
                                .catch(exception => {
                                  result = false
                                  showError(getErrorMessage(exception), 0, getString('pr.failedToSaveComment'))
                                })
                              break

                            case CommentAction.UPDATE:
                              await updateComment({ text: value }, { pathParams: { id } })
                                .then(newComment => {
                                  updatedItem = activityToCommentItem(newComment)
                                })
                                .catch(exception => {
                                  result = false
                                  showError(getErrorMessage(exception), 0, getString('pr.failedToSaveComment'))
                                })
                              break
                          }

                          if (result) {
                            onCommentUpdate()
                          }

                          return [result, updatedItem]
                        }}
                        outlets={{
                          [CommentBoxOutletPosition.TOP_OF_FIRST_COMMENT]: isCodeComment(commentItems) && (
                            <CodeCommentHeader commentItems={commentItems} />
                          )
                        }}
                      />
                    )
                  })}

                  <CommentBox
                    fluid
                    getString={getString}
                    commentItems={[]}
                    currentUserName={currentUser.display_name}
                    resetOnSave
                    hideCancel
                    handleAction={async (_action, value) => {
                      let result = true
                      let updatedItem: CommentItem<TypesPullReqActivity> | undefined = undefined

                      await saveComment({ text: value })
                        .then((newComment: TypesPullReqActivity) => {
                          updatedItem = activityToCommentItem(newComment)
                          setNewComments([...newComments, newComment])
                          setCommentCreated(true)
                        })
                        .catch(exception => {
                          result = false
                          showError(getErrorMessage(exception), 0)
                        })

                      if (result) {
                        onCommentUpdate()
                      }

                      return [result, updatedItem]
                    }}
                  />
                </Layout.Vertical>
              </Container>
              <PullRequestSideBar reviewers={reviewers} />
            </Layout.Horizontal>
          </Container>
        </Layout.Vertical>
      </Container>
    </PullRequestTabContentWrapper>
  )
}

const DescriptionBox: React.FC<ConversationProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onCommentUpdate: refreshPullRequestMetadata
}) => {
  const [edit, setEdit] = useState(false)
  // const [updated, setUpdated] = useState(pullRequestMetadata.edited as number)
  const [originalContent, setOriginalContent] = useState(pullRequestMetadata.description as string)
  const [content, setContent] = useState(originalContent)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}`
  })
  const name = pullRequestMetadata.author?.display_name

  return (
    <Container className={css.box}>
      <Layout.Vertical spacing="medium">
        <Container>
          <Layout.Horizontal spacing="xsmall" style={{ alignItems: 'center' }}>
            <StringSubstitute
              str={getString('pr.authorCommentedPR')}
              vars={{
                author: (
                  <>
                    <Avatar name={name} size="small" hoverCard={false} />
                    <Text inline margin={{ right: 'xsmall' }}>
                      <strong>{name}</strong>
                    </Text>
                  </>
                ),
                time: (
                  <Text inline>
                    <ReactTimeago date={pullRequestMetadata.created as number} />
                  </Text>
                )
              }}
            />

            {/* <PipeSeparator height={8} />
            <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
            </Text> */}
            <FlexExpander />
            <OptionsMenuButton
              isDark={false}
              icon="Options"
              iconProps={{ size: 14 }}
              style={{ padding: '5px' }}
              items={[
                {
                  text: getString('edit'),
                  onClick: () => setEdit(true)
                }
              ]}
            />
          </Layout.Horizontal>
        </Container>
        <Container padding={{ left: 'small', bottom: 'small' }}>
          {(edit && (
            <MarkdownEditorWithPreview
              value={content}
              onSave={value => {
                const payload: OpenapiUpdatePullReqRequest = {
                  title: pullRequestMetadata.title,
                  description: value
                }
                mutate(payload)
                  .then(() => {
                    setContent(value)
                    setOriginalContent(value)
                    setEdit(false)
                    // setUpdated(Date.now())
                    refreshPullRequestMetadata()
                  })
                  .catch(exception => showError(getErrorMessage(exception), 0, getString('pr.failedToUpdate')))
              }}
              onCancel={() => {
                setContent(originalContent)
                setEdit(false)
              }}
              i18n={{
                placeHolder: getString('pr.enterDesc'),
                tabEdit: getString('write'),
                tabPreview: getString('preview'),
                save: getString('save'),
                cancel: getString('cancel')
              }}
              maxEditorHeight="400px"
            />
          )) || <MarkdownViewer source={content} />}
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

function isCodeComment(commentItems: CommentItem<TypesPullReqActivity>[]) {
  return (commentItems[0]?.payload?.payload as Unknown)?.type === CommentType.CODE_COMMENT
}

interface CodeCommentHeaderProps {
  commentItems: CommentItem<TypesPullReqActivity>[]
}

const CodeCommentHeader: React.FC<CodeCommentHeaderProps> = ({ commentItems }) => {
  if (isCodeComment(commentItems)) {
    const payload = commentItems[0]?.payload?.payload as PullRequestCodeCommentPayload

    return (
      <Container className={css.snapshot}>
        <Layout.Vertical>
          <Container className={css.title}>
            <Text inline className={css.fname}>
              {payload?.file_title}
            </Text>
          </Container>
          <Container className={css.snapshotContent}>
            <Container className="d2h-wrapper">
              <Container className="d2h-file-wrapper line-by-line-file-diff">
                <Container className="d2h-file-diff">
                  <Container className="d2h-code-wrapper">
                    <table className="d2h-diff-table" cellPadding="0px" cellSpacing="0px">
                      <tbody
                        className="d2h-diff-tbody"
                        dangerouslySetInnerHTML={{
                          __html: payload?.diff_html_snapshot || ''
                        }}></tbody>
                    </table>
                  </Container>
                </Container>
              </Container>
            </Container>
          </Container>
        </Layout.Vertical>
      </Container>
    )
  }
  return null
}

function isSystemComment(commentItems: CommentItem<TypesPullReqActivity>[]) {
  return commentItems[0].payload?.kind === 'system'
}

interface SystemBoxProps extends Pick<GitInfoProps, 'pullRequestMetadata'> {
  commentItems: CommentItem<TypesPullReqActivity>[]
}

const SystemBox: React.FC<SystemBoxProps> = ({ pullRequestMetadata, commentItems }) => {
  const { getString } = useStrings()
  const payload = commentItems[0].payload
  const type = payload?.type

  switch (type) {
    case CommentType.MERGE: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.box}>
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

    case CommentType.BRANCH_UPDATE: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.box}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text>
              <StringSubstitute
                str={getString('pr.prBranchPushInfo')}
                vars={{
                  user: <strong>{payload?.author?.display_name}</strong>,
                  commit: <strong>{(payload?.payload as Unknown)?.new}</strong>
                }}
              />
            </Text>
            <FlexExpander />
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
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.box}>
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
            <FlexExpander />
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
        <Container className={css.box}>
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
            <FlexExpander />
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
        <Text className={css.box}>
          <Icon name={CodeIcon.Commit} padding={{ right: 'small' }} />
          {type}
        </Text>
      )
    }
  }
}

function useAnimateNewCommentBox(
  commentCreated: boolean,
  setCommentCreated: React.Dispatch<React.SetStateAction<boolean>>
) {
  useEffect(() => {
    let timeoutId = 0

    if (commentCreated) {
      timeoutId = window.setTimeout(() => {
        const box = document.querySelector(`.${css.newCommentCreated}`)

        box?.scrollIntoView({ behavior: 'smooth', block: 'center' })

        timeoutId = window.setTimeout(() => {
          box?.classList.add(css.clear)
          timeoutId = window.setTimeout(() => setCommentCreated(false), 2000)
        }, 5000)
      }, 300)
    }

    return () => {
      clearTimeout(timeoutId)
    }
  }, [commentCreated, setCommentCreated])
}
