import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { ButtonProps, Container, FlexExpander, Layout, Select, SelectOption, Text, useToaster } from '@harness/uicore'
import { useGet, useMutate } from 'restful-react'
import { orderBy } from 'lodash-es'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { TypesPullReqActivity, TypesPullReq } from 'services/code'
import { CommentAction, CommentBox, CommentBoxOutletPosition, CommentItem } from 'components/CommentBox/CommentBox'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage, orderSortDate, ButtonRoleProps } from 'utils/Utils'
import { activityToCommentItem } from 'components/DiffViewer/DiffViewerUtils'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { CodeCommentStatusSelect } from 'components/CodeCommentStatusSelect/CodeCommentStatusSelect'
import { CodeCommentStatusButton } from 'components/CodeCommentStatusButton/CodeCommentStatusButton'
import { CodeCommentSecondarySaveButton } from 'components/CodeCommentSecondarySaveButton/CodeCommentSecondarySaveButton'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import { DescriptionBox } from './DescriptionBox'
import { PullRequestActionsBox } from './PullRequestActionsBox/PullRequestActionsBox'
import PullRequestSideBar from './PullRequestSideBar/PullRequestSideBar'
import { isCodeComment, isComment, isSystemComment } from '../PullRequestUtils'
import { CodeCommentHeader } from './CodeCommentHeader'
import { SystemComment } from './SystemComment'
import css from './Conversation.module.scss'

export interface ConversationProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  onCommentUpdate: () => void
  prHasChanged?: boolean
  showEditDescription?: boolean
  onCancelEditDescription: () => void
}

export const Conversation: React.FC<ConversationProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onCommentUpdate,
  prHasChanged,
  showEditDescription,
  onCancelEditDescription
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
  const { data: reviewers, refetch: refetchReviewers } = useGet<Unknown[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/reviewers`
  })
  const { showError } = useToaster()
  const [dateOrderSort, setDateOrderSort] = useState<boolean | 'desc' | 'asc'>(orderSortDate.ASC)
  const activityFilters = useActivityFilters()
  const [activityFilter, setActivityFilter] = useState<SelectOption>(activityFilters[0] as SelectOption)
  const activityBlocks = useMemo(() => {
    // Each block may have one or more activities which are grouped into it. For example, one comment block
    // contains a parent comment and multiple replied comments
    const blocks: CommentItem<TypesPullReqActivity>[][] = []

    // Determine all parent activities
    const parentActivities = orderBy(
      activities?.filter(activity => !activity.parent_id) || [],
      'created',
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

    switch (activityFilter.value) {
      case PRCommentFilterType.ALL_COMMENTS:
        return blocks.filter(_activities => !isSystemComment(_activities))

      case PRCommentFilterType.RESOLVED_COMMENTS:
        return blocks.filter(
          _activities => _activities[0].payload?.resolved && (isCodeComment(_activities) || isComment(_activities))
        )

      case PRCommentFilterType.UNRESOLVED_COMMENTS:
        return blocks.filter(
          _activities => !_activities[0].payload?.resolved && (isComment(_activities) || isCodeComment(_activities))
        )

      case PRCommentFilterType.MY_COMMENTS: {
        const allCommentBlock = blocks.filter(_activities => !isSystemComment(_activities))
        const userCommentsOnly = allCommentBlock.filter(_activities => {
          const userCommentReply = _activities.filter(
            authorIsUser => authorIsUser.payload?.author?.uid === currentUser.uid
          )
          return userCommentReply.length !== 0
        })
        return userCommentsOnly
      }
    }

    return blocks
  }, [activities, dateOrderSort, activityFilter, currentUser.uid])
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
  const hasDescription = useMemo(
    () => !!pullRequestMetadata?.description?.length,
    [pullRequestMetadata?.description?.length]
  )

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
            refetchReviewers={refetchReviewers}
          />
          <Container>
            <Layout.Horizontal>
              <Container width={`70%`}>
                <Layout.Vertical spacing="xlarge">
                  {(hasDescription || showEditDescription) && (
                    <DescriptionBox
                      repoMetadata={repoMetadata}
                      pullRequestMetadata={pullRequestMetadata}
                      onCommentUpdate={onCommentUpdate}
                      onCancelEditDescription={onCancelEditDescription}
                    />
                  )}
                  <Layout.Horizontal
                    className={css.sortContainer}
                    padding={{ top: hasDescription || showEditDescription ? 'xxlarge' : undefined, bottom: 'medium' }}>
                    <Container>
                      <Select
                        items={activityFilters}
                        value={activityFilter}
                        className={css.selectButton}
                        onChange={newState => {
                          setActivityFilter(newState)
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
                              repoMetadataPath={repoMetadata.path}
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
                                  refetchActivities={refetchActivities}
                                />
                              ),
                              [CommentBoxOutletPosition.LEFT_OF_REPLY_PLACEHOLDER]: (
                                <CodeCommentStatusButton
                                  repoMetadata={repoMetadata}
                                  pullRequestMetadata={pullRequestMetadata}
                                  onCommentUpdate={onCommentUpdate}
                                  commentItems={commentItems}
                                />
                              ),
                              [CommentBoxOutletPosition.BETWEEN_SAVE_AND_CANCEL_BUTTONS]: (props: ButtonProps) => (
                                <CodeCommentSecondarySaveButton
                                  repoMetadata={repoMetadata}
                                  pullRequestMetadata={pullRequestMetadata as TypesPullReq}
                                  commentItems={commentItems}
                                  {...props}
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
              {repoMetadata ? (
                <PullRequestSideBar
                  reviewers={reviewers}
                  repoMetadata={repoMetadata}
                  pullRequestMetadata={pullRequestMetadata}
                  refetchReviewers={refetchReviewers}
                />
              ) : null}
            </Layout.Horizontal>
          </Container>
        </Layout.Vertical>
      </Container>
      <NavigationCheck when={dirtyCurrentComments || dirtyNewComment} />
    </PullRequestTabContentWrapper>
  )
}

export enum PRCommentFilterType {
  SHOW_EVERYTHING = 'showEverything',
  ALL_COMMENTS = 'allComments',
  MY_COMMENTS = 'myComments',
  RESOLVED_COMMENTS = 'resolvedComments',
  UNRESOLVED_COMMENTS = 'unresolvedComments'
}

function useActivityFilters() {
  const { getString } = useStrings()

  return useMemo(
    () => [
      {
        label: getString('showEverything'),
        value: PRCommentFilterType.SHOW_EVERYTHING
      },
      {
        label: getString('allComments'),
        value: PRCommentFilterType.ALL_COMMENTS
      },
      {
        label: getString('myComments'),
        value: PRCommentFilterType.MY_COMMENTS
      },
      {
        label: getString('unrsolvedComment'),
        value: PRCommentFilterType.UNRESOLVED_COMMENTS
      },
      {
        label: getString('resolvedComments'),
        value: PRCommentFilterType.RESOLVED_COMMENTS
      }
    ],
    [getString]
  )
}
