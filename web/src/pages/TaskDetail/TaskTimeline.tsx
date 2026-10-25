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
import { Link } from 'react-router-dom'
import { Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useAppContext } from 'AppContext'
import { StringKeys, useStrings } from 'framework/strings'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { TaskAssigneeInput, TypesTaskActivity, formatConfidence } from 'utils/TaskUtils'
import css from './TaskDetail.module.scss'

interface TaskTimelineProps {
  repoPath: string
  activities: TypesTaskActivity[]
}

/* eslint-disable @typescript-eslint/no-explicit-any */
export function TaskTimeline({ repoPath, activities }: TaskTimelineProps) {
  const { getString } = useStrings()
  const { routes } = useAppContext()

  const stateName = (state?: string) => (state ? getString(`tasks.stateValue.${state}` as StringKeys) : '-')
  const assigneeName = (a?: TaskAssigneeInput) =>
    a?.is_agent
      ? getString('tasks.agent')
      : a?.principal_id
      ? getString('tasks.principal', { id: a.principal_id })
      : a?.group_id
      ? getString('tasks.group', { id: a.group_id })
      : getString('tasks.unassigned')

  const renderDecision = (decision: any, key?: number) => (
    <Text key={key} className={css.decision}>
      {getString('tasks.activity.decisionLine', {
        question: decision?.question_id,
        value: decision?.value,
        confidence: formatConfidence(decision?.confidence)
      })}
      {decision?.fallback ? ` [${getString('tasks.activity.fallback')}]` : ''}
    </Text>
  )

  const renderBody = (activity: TypesTaskActivity): React.ReactNode => {
    const p = activity.payload || {}
    switch (activity.kind) {
      case 'created':
        return getString('tasks.activity.created', { title: p.title })
      case 'state_change':
        return (
          <>
            {getString('tasks.activity.stateChange', { from: stateName(p.from), to: stateName(p.to) })}
            {p.reason && <div className={css.quote}>{getString('tasks.activity.reason', { reason: p.reason })}</div>}
          </>
        )
      case 'decision':
        return (
          <>
            <strong>{getString('tasks.activity.decision')}</strong>
            {Array.isArray(p.decisions) ? (
              <>
                {p.summary && <div>{p.summary}</div>}
                {p.decisions.map((d: any, i: number) => renderDecision(d, i))}
                {p.outcome && <div>{p.outcome}</div>}
                {p.reason && <div className={css.quote}>{p.reason}</div>}
                {p.second_opinion && <div className={css.quote}>{p.second_opinion}</div>}
              </>
            ) : (
              renderDecision(p)
            )}
          </>
        )
      case 'agent_run':
        return getString('tasks.activity.agentRun', {
          outcome: p.outcome,
          model: p.model,
          requests: p.requests_used ?? 0
        })
      case 'pr_link': {
        const unlinked = p.action === 'unlinked'
        if (p.kind === 'pullreq') {
          return (
            <>
              {getString(unlinked ? 'tasks.activity.prUnlinked' : 'tasks.activity.prLinked', { id: p.ref_id })}{' '}
              <Link to={routes.toCODEPullRequest({ repoPath, pullRequestId: String(p.ref_id) })}>#{p.ref_id}</Link>
            </>
          )
        }
        return getString(unlinked ? 'tasks.activity.unlinked' : 'tasks.activity.linked', {
          kind: p.kind ? getString(`tasks.linkKind.${p.kind}` as StringKeys) : '',
          id: p.ref_id
        })
      }
      case 'comment':
        return (
          <>
            {getString('tasks.activity.commented')}
            <div className={css.quote}>{p.text}</div>
          </>
        )
      case 'assignment':
        return getString('tasks.activity.assignment', { from: assigneeName(p.from), to: assigneeName(p.to) })
      case 'priority_change':
        return getString('tasks.activity.priorityChange', {
          from: String(p.from || '').toUpperCase(),
          to: String(p.to || '').toUpperCase()
        })
      case 'updated':
        return getString('tasks.activity.updated', { fields: Object.keys(p.changes || {}).join(', ') })
      default:
        return activity.kind
    }
  }

  return (
    <ul className={css.timeline}>
      {activities.map(activity => (
        <li key={activity.id} className={css.timelineItem}>
          <Text color={Color.GREY_500} font={{ size: 'small' }}>
            {activity.principal_id
              ? getString('tasks.principal', { id: activity.principal_id })
              : getString('tasks.system')}
            {' · '}
            {activity.created ? <TimePopoverWithLocal time={activity.created} inline /> : null}
          </Text>
          <div>{renderBody(activity)}</div>
        </li>
      ))}
    </ul>
  )
}
