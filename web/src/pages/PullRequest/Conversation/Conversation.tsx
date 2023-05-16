import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Container, FlexExpander, Layout, Select, SelectOption, Text, useToaster } from '@harness/uicore'
import { useGet, useMutate } from 'restful-react'
import { orderBy } from 'lodash-es'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { TypesPullReqActivity } from 'services/code'
import { CommentAction, CommentBox, CommentBoxOutletPosition, CommentItem } from 'components/CommentBox/CommentBox'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage, orderSortDate, dayAgoInMS, ButtonRoleProps } from 'utils/Utils'
import { activityToCommentItem } from 'components/DiffViewer/DiffViewerUtils'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { CodeCommentStatusSelect } from 'components/CodeCommentStatusSelect/CodeCommentStatusSelect'
import { CodeCommentStatusButton } from 'components/CodeCommentStatusButton/CodeCommentStatusButton'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import { DescriptionBox } from './DescriptionBox'
import { PullRequestActionsBox } from './PullRequestActionsBox/PullRequestActionsBox'
import PullRequestSideBar from './PullRequestSideBar/PullRequestSideBar'
import { isCodeComment, isSystemComment } from '../PullRequestUtils'
import { CodeCommentHeader } from './CodeCommentHeader'
import { SystemComment } from './SystemComment'
import css from './Conversation.module.scss'

export interface ConversationProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  onCommentUpdate: () => void
  prHasChanged?: boolean
}

export enum prSortState {
  SHOW_EVERYTHING = 'showEverything',
  ALL_COMMENTS = 'allComments',
  WHATS_NEW = 'whatsNew',
  MY_COMMENTS = 'myComments'
}

export const Conversation: React.FC<ConversationProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onCommentUpdate,
  prHasChanged
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
  const showSpinner = useMemo(() => loading && !activities, [loading, activities])
  const { data: reviewers } = useGet<Unknown[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/reviewers`
  })
  const { showError } = useToaster()
  const [dateOrderSort, setDateOrderSort] = useState<boolean | 'desc' | 'asc'>(orderSortDate.ASC)
  const [prShowState, setPrShowState] = useState<SelectOption>({
    label: `Show Everything `,
    value: 'showEverything'
  })
  const activityBlocks = useMemo(() => {
    // Each block may have one or more activities which are grouped into it. For example, one comment block
    // contains a parent comment and multiple replied comments
    const blocks: CommentItem<TypesPullReqActivity>[][] = []

    // Determine all parent activities
    const parentActivities = orderBy(
      activities?.filter(activity => !activity.parent_id) || [],
      'edited',
      dateOrderSort
    ).map(_comment => [_comment])

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
    // return blocks.filter(_activities => !titleChangeItems.includes (_activities))

    if (prShowState.value === prSortState.ALL_COMMENTS) {
      const allCommentBlock = blocks.filter(_activities => !isSystemComment(_activities))
      return allCommentBlock
    }

    if (prShowState.value === prSortState.WHATS_NEW) {
      // get current time in seconds and subtract it by a day and see if comments are newer than a day
      const lastComment = blocks[blocks.length - 1]
      const lastCommentTime = lastComment[lastComment.length - 1].payload?.edited
      if (lastCommentTime !== undefined) {
        const currentTime = lastCommentTime - dayAgoInMS

        const newestBlock = blocks.filter(_activities => {
          const mostRecentComment = _activities[_activities.length - 1]
          if (mostRecentComment?.payload?.edited !== undefined) {
            return mostRecentComment?.payload?.edited > currentTime
          }
        })
        return newestBlock
      }
    }

    // show only comments made by user or replies in threads by user
    if (prShowState.value === prSortState.MY_COMMENTS) {
      const allCommentBlock = blocks.filter(_activities => !isSystemComment(_activities))
      const userCommentsOnly = allCommentBlock.filter(_activities => {
        const userCommentReply = _activities.filter(
          authorIsUser => authorIsUser.payload?.author?.uid === currentUser.uid
        )
        if (userCommentReply.length !== 0) {
          return true
        } else {
          return false
        }
      })
      return userCommentsOnly
    }

    return blocks
  }, [activities, dateOrderSort, prShowState, currentUser.uid])
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/comments`,
    [repoMetadata.path, pullRequestMetadata.number]
  )
  const { mutate: saveComment } = useMutate({ verb: 'POST', path })
  const { mutate: updateComment } = useMutate({ verb: 'PATCH', path: ({ id }) => `${path}/${id}` })
  const { mutate: deleteComment } = useMutate({ verb: 'DELETE', path: ({ id }) => `${path}/${id}` })
  const confirmAct = useConfirmAct()
  const [dirtyNewComment, setDirtyNewComment] = useState(false)
  const [dirtyCurrentComments, setDirtyCurrentComments] = useState(false)
  const onPRStateChanged = useCallback(() => {
    onCommentUpdate()
    refetchActivities()
  }, [onCommentUpdate, refetchActivities])

  useEffect(() => {
    if (prHasChanged) {
      refetchActivities()
    }
  }, [prHasChanged, refetchActivities])

  return (
    <PullRequestTabContentWrapper loading={showSpinner} error={error} onRetry={refetchActivities}>
      <Container>
        <Layout.Vertical spacing="xlarge">
          <PullRequestActionsBox
            repoMetadata={repoMetadata}
            pullRequestMetadata={pullRequestMetadata}
            onPRStateChanged={onPRStateChanged}
          />
          <Container>
            <Layout.Horizontal>
              <Container width={`70%`}>
                <Layout.Vertical spacing="xlarge">
                  <DescriptionBox
                    repoMetadata={repoMetadata}
                    pullRequestMetadata={pullRequestMetadata}
                    onCommentUpdate={onCommentUpdate}
                  />
                  <Layout.Horizontal className={css.sortContainer} padding={{ top: 'xxlarge', bottom: 'medium' }}>
                    <Container>
                      <Select
                        items={[
                          {
                            label: getString('showEverything'),
                            value: prSortState.SHOW_EVERYTHING
                          },
                          {
                            label: getString('allComments'),
                            value: prSortState.ALL_COMMENTS
                          },
                          {
                            label: getString('whatsNew'),
                            value: prSortState.WHATS_NEW
                          },
                          {
                            label: getString('myComments'),
                            value: prSortState.MY_COMMENTS
                          }
                        ]}
                        value={prShowState}
                        className={css.selectButton}
                        onChange={newState => {
                          setPrShowState(newState)
                          refetchActivities()
                        }}
                      />
                    </Container>
                    <FlexExpander />
                    <Text
                      {...ButtonRoleProps}
                      className={css.timeButton}
                      rightIconProps={{ size: 24 }}
                      rightIcon={dateOrderSort === orderSortDate.ASC ? 'code-ascending' : 'code-descending'}
                      onClick={() => {
                        if (dateOrderSort === orderSortDate.ASC) {
                          setDateOrderSort(orderSortDate.DESC)
                        } else {
                          setDateOrderSort(orderSortDate.ASC)
                        }
                      }}>
                      {dateOrderSort === orderSortDate.ASC ? getString('ascending') : getString('descending')}
                    </Text>
                  </Layout.Horizontal>

                  {activityBlocks?.map((commentItems, index) => {
                    const threadId = commentItems[0].payload?.id

                    if (isSystemComment(commentItems)) {
                      return (
                        <ThreadSection
                          key={`thread-${threadId}`}
                          onlyTitle
                          lastItem={activityBlocks.length - 1 === index}
                          title={
                            <SystemComment
                              key={`system-${threadId}`}
                              pullRequestMetadata={pullRequestMetadata}
                              commentItems={commentItems}
                            />
                          }></ThreadSection>
                      )
                    }
                    return (
                      <ThreadSection
                        key={`comment-${threadId}`}
                        onlyTitle={
                          activityBlocks[index + 1] !== undefined && isSystemComment(activityBlocks[index + 1])
                            ? true
                            : false
                        }
                        inCommentBox={
                          activityBlocks[index + 1] !== undefined && isSystemComment(activityBlocks[index + 1])
                            ? true
                            : false
                        }
                        title={
                          <CommentBox
                            key={threadId}
                            fluid
                            className={css.hideDottedLine}
                            commentItems={commentItems}
                            currentUserName={currentUser.display_name}
                            setDirty={setDirtyCurrentComments}
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
                                          showError(
                                            getErrorMessage(exception),
                                            0,
                                            getString('pr.failedToDeleteComment')
                                          )
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
                                <CodeCommentHeader commentItems={commentItems} threadId={threadId} />
                              ),
                              [CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]: (
                                <CodeCommentStatusSelect
                                  repoMetadata={repoMetadata}
                                  pullRequestMetadata={pullRequestMetadata}
                                  onCommentUpdate={onCommentUpdate}
                                  commentItems={commentItems}
                                />
                              ),

                              [CommentBoxOutletPosition.LEFT_OF_REPLY_PLACEHOLDER]: (
                                <CodeCommentStatusButton
                                  repoMetadata={repoMetadata}
                                  pullRequestMetadata={pullRequestMetadata}
                                  onCommentUpdate={onCommentUpdate}
                                  commentItems={commentItems}
                                />
                              )
                            }}
                            autoFocusAndPositioning
                          />
                        }></ThreadSection>
                    )
                  })}

                  <CommentBox
                    fluid
                    commentItems={[]}
                    currentUserName={currentUser.display_name}
                    resetOnSave
                    hideCancel
                    setDirty={setDirtyNewComment}
                    handleAction={async (_action, value) => {
                      let result = true
                      let updatedItem: CommentItem<TypesPullReqActivity> | undefined = undefined

                      await saveComment({ text: value })
                        .then((newComment: TypesPullReqActivity) => {
                          updatedItem = activityToCommentItem(newComment)
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
      <NavigationCheck when={dirtyCurrentComments || dirtyNewComment} />
    </PullRequestTabContentWrapper>
  )
}
