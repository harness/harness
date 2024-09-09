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

import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  ButtonProps,
  Container,
  FlexExpander,
  Layout,
  Select,
  SelectOption,
  Text,
  Utils,
  useToaster
} from '@harnessio/uicore'
import { useLocation } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import { get, orderBy } from 'lodash-es'
import { Render } from 'react-jsx-match'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type {
  TypesPullReqActivity,
  TypesPullReq,
  TypesPullReqStats,
  TypesCodeOwnerEvaluation,
  TypesPullReqReviewer,
  TypesListCommitResponse,
  TypesScopesLabels
} from 'services/code'
import { CommentAction, CommentBox, CommentBoxOutletPosition, CommentItem } from 'components/CommentBox/CommentBox'
import { useConfirmAct } from 'hooks/useConfirmAction'
import {
  getErrorMessage,
  orderSortDate,
  ButtonRoleProps,
  PullRequestSection,
  filenameToLanguage,
  PRCommentFilterType
} from 'utils/Utils'
import { CommentType, activitiesToDiffCommentItems, activityToCommentItem } from 'components/DiffViewer/DiffViewerUtils'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { CodeCommentStatusSelect } from 'components/CodeCommentStatusSelect/CodeCommentStatusSelect'
import { CodeCommentStatusButton } from 'components/CodeCommentStatusButton/CodeCommentStatusButton'
import { CodeCommentSecondarySaveButton } from 'components/CodeCommentSecondarySaveButton/CodeCommentSecondarySaveButton'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { CommentThreadTopDecoration } from 'components/CommentThreadTopDecoration/CommentThreadTopDecoration'
import { getConfig } from 'services/config'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import { DescriptionBox } from './DescriptionBox'
import PullRequestSideBar from './PullRequestSideBar/PullRequestSideBar'
import { isCodeComment, isComment, isSystemComment } from '../PullRequestUtils'
import { usePullReqActivities } from '../useGetPullRequestInfo'
import { CodeCommentHeader } from './CodeCommentHeader'
import { SystemComment } from './SystemComment'
import PullRequestOverviewPanel from './PullRequestOverviewPanel/PullRequestOverviewPanel'
import css from './Conversation.module.scss'

export interface ConversationProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  prStats?: TypesPullReqStats
  showEditDescription?: boolean
  onCancelEditDescription: () => void
  onDescriptionSaved: () => void
  prChecksDecisionResult?: PRChecksDecisionResult
  standalone: boolean
  routingId: string
  pullReqCommits?: TypesListCommitResponse
  refetchActivities: () => void
  refetchPullReq: () => void
}

export const Conversation: React.FC<ConversationProps> = ({
  repoMetadata,
  pullReqMetadata,
  onDescriptionSaved,
  prStats,
  showEditDescription,
  onCancelEditDescription,
  prChecksDecisionResult,
  standalone,
  routingId,
  pullReqCommits,
  refetchActivities,
  refetchPullReq
}) => {
  const { getString } = useStrings()
  const { currentUser, routes, hooks } = useAppContext()
  const { CODE_PULLREQ_LABELS: isLabelEnabled } = hooks?.useFeatureFlags()
  const location = useLocation()
  const activities = usePullReqActivities()
  const {
    data: reviewers,
    refetch: refetchReviewers,
    loading: loadingReviewers
  } = useGet<TypesPullReqReviewer[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/reviewers`,
    debounce: 500
  })

  const { data: labels, refetch: refetchLabels } = useGet<TypesScopesLabels>({
    base: getConfig('code/api/v1'),
    path: `/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/labels`,
    debounce: 500
  })

  const { data: codeOwners, refetch: refetchCodeOwners } = useGet<TypesCodeOwnerEvaluation>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/codeowners`,
    debounce: 500
  })

  const { showError } = useToaster()
  const [dateOrderSort, setDateOrderSort] = useUserPreference<orderSortDate.ASC | orderSortDate.DESC>(
    UserPreference.PULL_REQUEST_ACTIVITY_ORDER,
    orderSortDate.ASC
  )
  const activityFilters = useActivityFilters()
  const [activityFilter, setActivityFilter] = useUserPreference<SelectOption>(
    UserPreference.PULL_REQUEST_ACTIVITY_FILTER,
    activityFilters[0] as SelectOption
  )

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
        return blocks?.filter(_activities => !isSystemComment(_activities))

      case PRCommentFilterType.RESOLVED_COMMENTS:
        return blocks?.filter(
          _activities => _activities[0].payload?.resolved && (isCodeComment(_activities) || isComment(_activities))
        )

      case PRCommentFilterType.UNRESOLVED_COMMENTS:
        return blocks?.filter(
          _activities => !_activities[0].payload?.resolved && (isComment(_activities) || isCodeComment(_activities))
        )

      case PRCommentFilterType.MY_COMMENTS: {
        const allCommentBlock = blocks?.filter(_activities => !isSystemComment(_activities))
        const userCommentsOnly = allCommentBlock?.filter(_activities => {
          const userCommentReply = _activities?.filter(
            authorIsUser => currentUser?.uid && authorIsUser.payload?.author?.uid === currentUser?.uid
          )
          return userCommentReply.length !== 0
        })
        return userCommentsOnly
      }
    }

    return blocks
  }, [activities, dateOrderSort, activityFilter, currentUser?.uid])
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/comments`,
    [repoMetadata.path, pullReqMetadata.number]
  )
  const { mutate: saveComment } = useMutate({ verb: 'POST', path })
  const { mutate: updateComment } = useMutate({ verb: 'PATCH', path: ({ id }) => `${path}/${id}` })
  const { mutate: deleteComment } = useMutate({ verb: 'DELETE', path: ({ id }) => `${path}/${id}` })
  const confirmAct = useConfirmAct()
  const [dirtyNewComment, setDirtyNewComment] = useState(false)
  const [dirtyCurrentComments, setDirtyCurrentComments] = useState(false)
  const onPRStateChanged = useCallback(() => {
    refetchCodeOwners()
  }, [refetchCodeOwners])
  const hasDescription = useMemo(() => !!pullReqMetadata?.description?.length, [pullReqMetadata?.description?.length])
  const copyLinkToComment = useCallback(
    (id, commentItem) => {
      const _path = `${routes.toCODEPullRequest({
        repoPath: repoMetadata?.path as string,
        pullRequestId: String(pullReqMetadata?.number),
        pullRequestSection: PullRequestSection.FILES_CHANGED
      })}?path=${commentItem.payload?.code_comment?.path}&commentId=${id}`
      const { pathname, origin } = window.location

      Utils.copy(origin + pathname.replace(location.pathname, '') + _path)
    },
    [location, pullReqMetadata?.number, repoMetadata?.path, routes]
  )

  useEffect(() => {
    if (prStats) {
      refetchCodeOwners()
    }
  }, [
    prStats,
    prStats?.conversations,
    prStats?.unresolved_count,
    pullReqMetadata?.title,
    pullReqMetadata?.state,
    pullReqMetadata?.source_sha,
    refetchCodeOwners
  ])

  const newCommentBox = useMemo(() => {
    return (
      <CommentBox
        routingId={routingId}
        standalone={standalone}
        repoMetadata={repoMetadata}
        fluid
        boxClassName={css.commentBox}
        editorClassName={css.commentEditor}
        commentItems={[]}
        currentUserName={currentUser?.display_name || currentUser?.email || ''}
        resetOnSave
        hideCancel={false}
        setDirty={setDirtyNewComment}
        enableReplyPlaceHolder={true}
        autoFocusAndPosition={true}
        copyLinkToComment={copyLinkToComment}
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

          return [result, updatedItem]
        }}
      />
    ) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentUser, saveComment, showError])

  const renderedActivityBlocks = useMemo(
    () =>
      activityBlocks?.map((commentItems, index) => {
        const threadId = commentItems[0].payload?.id
        const renderLabelActivities =
          commentItems[0].payload?.type !== CommentType.LABEL_MODIFY || isLabelEnabled || standalone
        if (isSystemComment(commentItems)) {
          return (
            <Render key={`thread-${threadId}`} when={renderLabelActivities}>
              <ThreadSection
                key={`thread-${threadId}`}
                onlyTitle
                lastItem={activityBlocks.length - 1 === index}
                title={
                  <SystemComment
                    key={`system-${threadId}`}
                    pullReqMetadata={pullReqMetadata}
                    commentItems={commentItems}
                    repoMetadataPath={repoMetadata.path}
                  />
                }></ThreadSection>
            </Render>
          )
        }

        const activity = commentItems[0].payload
        const right = get(activity?.payload, 'line_start_new', false)
        const span = right ? activity?.code_comment?.span_new || 0 : activity?.code_comment?.span_old || 0
        const startLine = (right ? activity?.code_comment?.line_new : activity?.code_comment?.line_old) as number
        const endLine = startLine + span - 1

        const comment = activitiesToDiffCommentItems(activity?.code_comment?.path as string, [
          activity as TypesPullReqActivity
        ])[0]

        const suggestionBlock = comment.left
          ? undefined
          : {
              source: comment.codeBlockContent as string,
              lang: filenameToLanguage(activity?.code_comment?.path?.split('/').pop())
            }

        return (
          <ThreadSection
            key={`comment-${threadId}-${activity?.created}-${activity?.edited}-${activity?.resolved}-${activity?.code_comment?.outdated}`}
            onlyTitle={
              activityBlocks[index + 1] !== undefined && isSystemComment(activityBlocks[index + 1]) ? true : false
            }
            inCommentBox={
              activityBlocks[index + 1] !== undefined && isSystemComment(activityBlocks[index + 1]) ? true : false
            }
            title={
              <CommentBox
                routingId={routingId}
                standalone={standalone}
                repoMetadata={repoMetadata}
                key={threadId}
                fluid
                boxClassName={css.threadbox}
                outerClassName={css.hideDottedLine}
                commentItems={commentItems}
                currentUserName={currentUser?.display_name || currentUser?.email || ''}
                setDirty={setDirtyCurrentComments}
                enableReplyPlaceHolder={true}
                autoFocusAndPosition={true}
                copyLinkToComment={copyLinkToComment}
                suggestionBlock={suggestionBlock}
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

                  return [result, updatedItem]
                }}
                outlets={{
                  [CommentBoxOutletPosition.TOP_OF_FIRST_COMMENT]: isCodeComment(commentItems) && (
                    <>
                      <CommentThreadTopDecoration startLine={startLine} endLine={endLine} />
                      <CodeCommentHeader
                        commentItems={commentItems}
                        threadId={threadId}
                        repoMetadata={repoMetadata}
                        pullReqMetadata={pullReqMetadata}
                      />
                    </>
                  ),
                  [CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]: (
                    <CodeCommentStatusSelect
                      repoMetadata={repoMetadata}
                      pullReqMetadata={pullReqMetadata}
                      comment={{ commentItems }}
                    />
                  ),
                  [CommentBoxOutletPosition.RIGHT_OF_REPLY_PLACEHOLDER]: (
                    <CodeCommentStatusButton
                      repoMetadata={repoMetadata}
                      pullReqMetadata={pullReqMetadata}
                      comment={{ commentItems }}
                    />
                  ),
                  [CommentBoxOutletPosition.BETWEEN_SAVE_AND_CANCEL_BUTTONS]: (props: ButtonProps) => (
                    <CodeCommentSecondarySaveButton
                      repoMetadata={repoMetadata}
                      pullReqMetadata={pullReqMetadata as TypesPullReq}
                      comment={{ commentItems }}
                      {...props}
                    />
                  )
                }}
              />
            }></ThreadSection>
        )
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [activityBlocks, currentUser, pullReqMetadata, activities]
  )

  return (
    <PullRequestTabContentWrapper section={PullRequestSection.CONVERSATION}>
      <Container>
        <Layout.Vertical spacing="xlarge">
          <Container>
            <Layout.Horizontal width="calc(var(--page-container-width) - 48px)">
              <Container width={`70%`}>
                <Layout.Vertical spacing="xlarge">
                  {prChecksDecisionResult && (
                    <Container padding={{ bottom: 'small' }}>
                      <PullRequestOverviewPanel
                        repoMetadata={repoMetadata}
                        pullReqMetadata={pullReqMetadata}
                        onPRStateChanged={onPRStateChanged}
                        refetchReviewers={refetchReviewers}
                        prChecksDecisionResult={prChecksDecisionResult}
                        codeOwners={codeOwners}
                        reviewers={reviewers}
                        pullReqCommits={pullReqCommits}
                        setActivityFilter={setActivityFilter}
                        loadingReviewers={loadingReviewers}
                        refetchActivities={refetchActivities}
                        refetchCodeOwners={refetchCodeOwners}
                        refetchPullReq={refetchPullReq}
                        activities={activities}
                      />
                    </Container>
                  )}
                  {(hasDescription || showEditDescription) && (
                    <DescriptionBox
                      routingId={routingId}
                      standalone={standalone}
                      repoMetadata={repoMetadata}
                      pullReqMetadata={pullReqMetadata}
                      onDescriptionSaved={onDescriptionSaved}
                      onCancelEditDescription={onCancelEditDescription}
                      prStats={prStats}
                      pullReqCommits={pullReqCommits}
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

                  {dateOrderSort != orderSortDate.DESC ? null : (
                    <Container className={css.descContainer}>{newCommentBox}</Container>
                  )}

                  {renderedActivityBlocks}

                  {dateOrderSort != orderSortDate.ASC ? null : (
                    <Container className={css.ascContainer}>{newCommentBox}</Container>
                  )}
                </Layout.Vertical>
              </Container>

              <PullRequestSideBar
                reviewers={reviewers}
                repoMetadata={repoMetadata}
                pullRequestMetadata={pullReqMetadata}
                refetchReviewers={refetchReviewers}
                labels={labels}
                refetchLabels={refetchLabels}
                refetchActivities={refetchActivities}
              />
            </Layout.Horizontal>
          </Container>
        </Layout.Vertical>
      </Container>
      <NavigationCheck when={dirtyCurrentComments || dirtyNewComment} />
    </PullRequestTabContentWrapper>
  )
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
