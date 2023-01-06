import React, { useMemo, useState } from 'react'
import { Avatar, Color, Container, FlexExpander, FontVariation, Layout, Text, useToaster } from '@harness/uicore'
import { useGet, useMutate } from 'restful-react'
import ReactTimeago from 'react-timeago'
import type { GitInfoProps } from 'utils/GitUtils'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { TypesPullReqActivity } from 'services/code'
import { CommentAction, CommentBox, CommentItem } from 'components/CommentBox/CommentBox'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage } from 'utils/Utils'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import { PullRequestStatusInfo } from './PullRequestStatusInfo/PullRequestStatusInfo'
import css from './Conversation.module.scss'

export const Conversation: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  const { getString } = useStrings()
  const { currentUser } = useAppContext()
  const {
    data: activities,
    loading,
    error,
    refetch
  } = useGet<TypesPullReqActivity[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.id}/activities`
  })
  const { showError } = useToaster()
  const [newComments, setNewComments] = useState<TypesPullReqActivity[]>([])
  const commentThreads = useMemo(() => {
    const threads: Record<number, CommentItem<TypesPullReqActivity>[]> = {}
    activities?.forEach(activity => {
      const thread: CommentItem<TypesPullReqActivity> = {
        author: activity.author?.name as string,
        created: activity.created as number,
        updated: activity.updated as number,
        deleted: activity.deleted as number,
        content: activity.text as string,
        payload: activity
      }

      if (activity.parent_id) {
        threads[activity.parent_id].push(thread)
      } else {
        threads[activity.id as number] = threads[activity.id as number] || []
        threads[activity.id as number].push(thread)
      }
    })
    newComments.forEach(newComment => {
      threads[newComment.id as number] = [
        {
          author: newComment.author?.name as string,
          created: newComment.created as number,
          updated: newComment.updated as number,
          deleted: newComment.deleted as number,
          content: newComment.text as string,
          payload: newComment
        }
      ]
    })
    return threads
  }, [activities, newComments])
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.id}/comments`,
    [repoMetadata.path, pullRequestMetadata.id]
  )
  const { mutate: saveComment } = useMutate({ verb: 'POST', path })
  const { mutate: updateComment } = useMutate({ verb: 'PUT', path: ({ id }) => `${path}/${id}` })
  const { mutate: deleteComment } = useMutate({ verb: 'DELETE', path: ({ id }) => `${path}/${id}` })
  const confirmAct = useConfirmAct()

  return (
    <PullRequestTabContentWrapper loading={loading} error={error} onRetry={refetch}>
      <Container padding="xsmall">
        <Layout.Vertical spacing="xlarge">
          <PullRequestStatusInfo />
          <Container>
            <Layout.Vertical spacing="xlarge">
              <DescriptionBox repoMetadata={repoMetadata} pullRequestMetadata={pullRequestMetadata} />

              {Object.entries(commentThreads).map(([threadId, commentItems]) => (
                <CommentBox
                  key={threadId}
                  fluid
                  getString={getString}
                  commentItems={commentItems}
                  currentUserName={currentUser.display_name}
                  handleAction={async (action, value, commentItem) => {
                    let result = true
                    const id = (commentItem as CommentItem<TypesPullReqActivity>)?.payload?.id

                    switch (action) {
                      case CommentAction.DELETE:
                        await confirmAct({
                          message: getString('deleteCommentConfirm'),
                          action: async () => {
                            await deleteComment({}, { pathParams: { id } }).catch(exception => {
                              result = false
                              showError(getErrorMessage(exception), 0, getString('pr.failedToDeleteComment'))
                            })
                          }
                        })
                        break
                      case CommentAction.REPLY:
                        await saveComment({ text: value, parent_id: Number(threadId) }).catch(exception => {
                          result = false
                          showError(getErrorMessage(exception), 0, getString('pr.failedToSaveComment'))
                        })
                        break
                      case CommentAction.UPDATE:
                        await updateComment({ text: value }, { pathParams: { id } }).catch(exception => {
                          result = false
                          showError(getErrorMessage(exception), 0, getString('pr.failedToSaveComment'))
                        })
                        break
                    }

                    return result
                  }}
                />
              ))}

              <CommentBox
                fluid
                getString={getString}
                commentItems={[]}
                currentUserName={currentUser.display_name}
                resetOnSave
                hideCancel
                handleAction={async (_action, value) => {
                  let result = true
                  await saveComment({ text: value })
                    .then((newComment: TypesPullReqActivity) => setNewComments([...newComments, newComment]))
                    .catch(exception => {
                      result = false
                      showError(getErrorMessage(exception), 0, getString('pr.failedToSaveComment'))
                    })
                  return result
                }}
              />
            </Layout.Vertical>
          </Container>
        </Layout.Vertical>
      </Container>
    </PullRequestTabContentWrapper>
  )
}

const DescriptionBox: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  const [edit, setEdit] = useState(false)
  const [updated, setUpdated] = useState(pullRequestMetadata.updated as number)
  const [originalContent, setOriginalContent] = useState(pullRequestMetadata.description as string)
  const [content, setContent] = useState(originalContent)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.id}`
  })

  return (
    <Container className={css.descBox}>
      <Layout.Vertical spacing="medium">
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={pullRequestMetadata.author?.name} size="small" hoverCard={false} />
            <Text inline>
              <strong>{pullRequestMetadata.author?.name}</strong>
            </Text>
            <PipeSeparator height={8} />
            <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
              <ReactTimeago date={updated} />
            </Text>
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
                mutate({
                  title: pullRequestMetadata.title,
                  description: value
                })
                  .then(() => {
                    setContent(value)
                    setOriginalContent(value)
                    setEdit(false)
                    setUpdated(Date.now())
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
