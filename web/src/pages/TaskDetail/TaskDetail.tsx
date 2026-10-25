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

import React, { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  DropDown,
  Layout,
  PageBody,
  Text,
  TextInput,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGet, useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { StringKeys, useStrings } from 'framework/strings'
import type { CODEProps } from 'RouteDefinitions'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TaskStateBadge } from 'components/TaskStateBadge/TaskStateBadge'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { getErrorMessage, voidFn } from 'utils/Utils'
import {
  TASK_PRIORITIES,
  TaskAssigneeInput,
  TaskPriority,
  TaskState,
  TaskUpdateInput,
  TypesTask,
  TypesTaskActivity,
  TypesTaskLink,
  dueToInputValue,
  getAllowedTransitions,
  getAssigneeLabel,
  getTaskApiPath,
  inputValueToDue
} from 'utils/TaskUtils'
import { TaskTimeline } from './TaskTimeline'
import css from './TaskDetail.module.scss'

export default function TaskDetail() {
  const { getString } = useStrings()
  const { routes, currentUser } = useAppContext()
  const { taskNumber } = useParams<CODEProps>()
  const { showError, showSuccess } = useToaster()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const repoPath = repoMetadata?.path as string
  const taskPath = `${getTaskApiPath(repoMetadata?.path)}/${taskNumber}`

  const [editing, setEditing] = useState(false)
  const [draftTitle, setDraftTitle] = useState('')
  const [draftDescription, setDraftDescription] = useState('')
  const [dismissOpen, setDismissOpen] = useState(false)
  const [dismissReason, setDismissReason] = useState('')
  const [comment, setComment] = useState('')

  const {
    data: task,
    error: taskError,
    loading: taskLoading,
    refetch: refetchTask
  } = useGet<TypesTask>({ path: taskPath, lazy: !repoMetadata })
  const { data: links, refetch: refetchLinks } = useGet<TypesTaskLink[]>({
    path: `${taskPath}/links`,
    lazy: !repoMetadata
  })
  const { data: activities, refetch: refetchActivities } = useGet<TypesTaskActivity[]>({
    path: `${taskPath}/activities`,
    lazy: !repoMetadata
  })

  const { mutate: patchTask } = useMutate<TypesTask>({ verb: 'PATCH', path: taskPath })
  const { mutate: postState } = useMutate<TypesTask>({ verb: 'POST', path: `${taskPath}/state` })
  const { mutate: postComment } = useMutate({ verb: 'POST', path: `${taskPath}/comments` })

  const refreshAll = () => {
    refetchTask()
    refetchLinks()
    refetchActivities()
  }

  const update = (input: TaskUpdateInput, onDone?: () => void) =>
    patchTask(input)
      .then(() => {
        showSuccess(getString('tasks.taskUpdated'))
        onDone?.()
        refreshAll()
      })
      .catch(err => showError(getErrorMessage(err)))

  const moveTo = (state: TaskState, reason?: string) =>
    postState({ state, reason })
      .then(() => {
        showSuccess(getString('tasks.stateChanged', { state: getString(`tasks.stateValue.${state}` as StringKeys) }))
        setDismissOpen(false)
        setDismissReason('')
        refreshAll()
      })
      .catch(err => showError(getErrorMessage(err)))

  const sendComment = () => {
    if (!comment.trim()) {
      return
    }
    postComment({ text: comment.trim() })
      .then(() => {
        showSuccess(getString('tasks.commentAdded'))
        setComment('')
        refetchActivities()
      })
      .catch(err => showError(getErrorMessage(err)))
  }

  const assigneeValue = task?.assignee_is_agent
    ? 'agent'
    : task?.assignee_principal_id && task.assignee_principal_id === currentUser?.id
    ? 'me'
    : task?.assignee_principal_id || task?.assignee_group_id
    ? 'other'
    : 'none'

  const assigneeItems = [
    { label: getString('tasks.unassigned'), value: 'none' },
    { label: getString('tasks.agent'), value: 'agent' },
    { label: getString('tasks.me'), value: 'me' },
    ...(assigneeValue === 'other' && task ? [{ label: getAssigneeLabel(task, getString), value: 'other' }] : [])
  ]

  const onAssigneeChange = (value: string) => {
    let assignee: TaskAssigneeInput | undefined
    if (value === 'none') {
      assignee = {}
    } else if (value === 'agent') {
      assignee = { is_agent: true }
    } else if (value === 'me') {
      assignee = { principal_id: currentUser?.id }
    }
    if (assignee && value !== assigneeValue) {
      update({ assignee })
    }
  }

  const startEdit = () => {
    setDraftTitle(task?.title || '')
    setDraftDescription(task?.description || '')
    setEditing(true)
  }

  // Links to the pages of the linked entities. Insights issues and Shield findings have their own pages.
  const renderLink = (link: TypesTaskLink) => {
    switch (link.kind) {
      case 'pullreq':
        return (
          <Link to={routes.toCODEPullRequest({ repoPath, pullRequestId: String(link.ref_id) })}>
            {getString('tasks.pullRequest', { id: link.ref_id })}
          </Link>
        )
      case 'insights_issue':
        return (
          <Link to={routes.toCODEError({ repoPath, issueId: String(link.ref_id) })}>
            {getString('tasks.insightsIssue', { id: link.ref_id })}
          </Link>
        )
      case 'shield_finding':
        return (
          <Link to={routes.toCODESecurity({ repoPath })}>{getString('tasks.shieldFinding', { id: link.ref_id })}</Link>
        )
      default:
        return (
          <Text>
            {getString(`tasks.linkKind.${link.kind}` as StringKeys)}: {link.ref_id}
          </Text>
        )
    }
  }

  const renderOrigin = () => {
    if (!task?.origin_ref) {
      return <Text>{getString('tasks.noOriginRef')}</Text>
    }
    const originLink = links?.find(
      link => link.kind === (task.origin === 'insights' ? 'insights_issue' : 'shield_finding')
    )
    return (
      <Text>
        {originLink ? (
          renderLink(originLink)
        ) : (
          <>
            {getString(`tasks.originValue.${task.origin}` as StringKeys)}: {task.origin_ref}
          </>
        )}
      </Text>
    )
  }

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={task ? `#${task.number} ${task.title}` : getString('pageTitle.taskDetail')}
        dataTooltipId="repositoryTask"
        extraBreadcrumbLinks={
          repoMetadata ? [{ label: getString('tasks.title'), url: routes.toCODETasks({ repoPath }) }] : undefined
        }
      />
      <PageBody error={getErrorMessage(error || taskError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || taskLoading} withBorder />
        <Render when={repoMetadata && task}>
          <Container padding="xlarge">
            <div className={css.layout}>
              <div className={css.content}>
                {editing ? (
                  <Layout.Vertical spacing="small">
                    <TextInput value={draftTitle} onChange={e => setDraftTitle((e.target as HTMLInputElement).value)} />
                    <textarea
                      className={css.textarea}
                      value={draftDescription}
                      onChange={e => setDraftDescription(e.target.value)}
                    />
                    <Layout.Horizontal spacing="small">
                      <Button
                        variation={ButtonVariation.PRIMARY}
                        text={getString('tasks.save')}
                        disabled={!draftTitle.trim()}
                        onClick={() =>
                          update({ title: draftTitle.trim(), description: draftDescription }, () => setEditing(false))
                        }
                      />
                      <Button
                        variation={ButtonVariation.TERTIARY}
                        text={getString('tasks.cancel')}
                        onClick={() => setEditing(false)}
                      />
                    </Layout.Horizontal>
                  </Layout.Vertical>
                ) : (
                  <>
                    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                      <Text font={{ variation: FontVariation.H3 }}>{task?.title}</Text>
                      <TaskStateBadge state={task?.state} />
                      <Button
                        variation={ButtonVariation.LINK}
                        icon="Edit"
                        text={getString('tasks.edit')}
                        onClick={startEdit}
                      />
                    </Layout.Horizontal>
                    <div className={css.description}>{task?.description}</div>
                  </>
                )}

                <Render when={task?.state_reason}>
                  <Text color={Color.GREY_500}>
                    {getString('tasks.activity.reason', { reason: task?.state_reason })}
                  </Text>
                </Render>

                <div className={css.actions}>
                  <Text>{getString('tasks.stateActions')}:</Text>
                  {getAllowedTransitions(task?.state).map(state => (
                    <Button
                      key={state}
                      variation={state === 'descartada' ? ButtonVariation.TERTIARY : ButtonVariation.SECONDARY}
                      text={getString(`tasks.stateValue.${state}` as StringKeys)}
                      onClick={() => (state === 'descartada' ? setDismissOpen(true) : moveTo(state))}
                    />
                  ))}
                </div>

                <Text font={{ variation: FontVariation.H5 }}>{getString('tasks.timeline')}</Text>
                <Container padding={{ top: 'medium', bottom: 'medium' }}>
                  {activities?.length ? (
                    <TaskTimeline repoPath={repoPath} activities={activities} />
                  ) : (
                    <Text>{getString('tasks.noActivity')}</Text>
                  )}
                </Container>

                <Layout.Vertical spacing="small">
                  <textarea
                    className={css.textarea}
                    placeholder={getString('tasks.addComment')}
                    value={comment}
                    onChange={e => setComment(e.target.value)}
                  />
                  <Container>
                    <Button
                      variation={ButtonVariation.PRIMARY}
                      text={getString('tasks.comment')}
                      disabled={!comment.trim()}
                      onClick={sendComment}
                    />
                  </Container>
                </Layout.Vertical>
              </div>

              <div className={css.sidebar}>
                <div className={css.field}>
                  <Text color={Color.GREY_500}>{getString('tasks.priority')}</Text>
                  <DropDown
                    value={task?.priority}
                    items={TASK_PRIORITIES.map(p => ({ label: p.toUpperCase(), value: p }))}
                    onChange={({ value }) => value !== task?.priority && update({ priority: value as TaskPriority })}
                  />
                </div>
                <div className={css.field}>
                  <Text color={Color.GREY_500}>{getString('tasks.assignee')}</Text>
                  <DropDown
                    value={assigneeValue}
                    items={assigneeItems}
                    onChange={({ value }) => onAssigneeChange(value as string)}
                  />
                </div>
                <div className={css.field}>
                  <Text color={Color.GREY_500}>{getString('tasks.due')}</Text>
                  <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                    <input
                      type="date"
                      value={dueToInputValue(task?.due)}
                      onChange={e => {
                        const due = inputValueToDue(e.target.value)
                        if (due) {
                          update({ due })
                        }
                      }}
                    />
                    <Render when={task?.due}>
                      <Button
                        variation={ButtonVariation.ICON}
                        icon="cross"
                        tooltip={getString('tasks.clearDue')}
                        onClick={() => update({ clear_due: true })}
                      />
                    </Render>
                  </Layout.Horizontal>
                </div>
                <div className={css.field}>
                  <Text color={Color.GREY_500}>{getString('tasks.origin')}</Text>
                  {renderOrigin()}
                </div>
                <div className={css.field}>
                  <Text color={Color.GREY_500}>{getString('tasks.links')}</Text>
                  {links?.length ? (
                    links.map(link => <div key={`${link.kind}-${link.ref_id}`}>{renderLink(link)}</div>)
                  ) : (
                    <Text>{getString('tasks.noLinks')}</Text>
                  )}
                </div>
                <Render when={task?.created}>
                  <Text color={Color.GREY_500}>
                    <TimePopoverWithLocal time={task?.created as number} inline />
                  </Text>
                </Render>
              </div>
            </div>
          </Container>
        </Render>
      </PageBody>

      <Dialog
        isOpen={dismissOpen}
        enforceFocus={false}
        onClose={() => setDismissOpen(false)}
        title={getString('tasks.dismissTitle')}>
        <Layout.Vertical spacing="medium">
          <Text>{getString('tasks.dismissReason')}</Text>
          <textarea
            className={css.textarea}
            value={dismissReason}
            onChange={e => setDismissReason(e.target.value)}
            autoFocus
          />
          {!dismissReason.trim() && <Text color={Color.GREY_500}>{getString('tasks.dismissReasonRequired')}</Text>}
          <Layout.Horizontal spacing="small">
            <Button
              variation={ButtonVariation.PRIMARY}
              text={getString('tasks.dismiss')}
              disabled={!dismissReason.trim()}
              onClick={() => moveTo('descartada', dismissReason.trim())}
            />
            <Button
              variation={ButtonVariation.TERTIARY}
              text={getString('tasks.cancel')}
              onClick={() => setDismissOpen(false)}
            />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Dialog>
    </Container>
  )
}
