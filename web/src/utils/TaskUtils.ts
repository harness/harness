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

import { useGet } from 'restful-react'
import type { StringKeys } from 'framework/strings'

// Types and helpers for the Tasks REST API (/api/v1/repos/{repo_ref}/tasks).
// The generated services do not include tasks yet, hence the manual types.

export type TaskState =
  | 'abierta'
  | 'triage'
  | 'en_curso'
  | 'en_revision'
  | 'resuelta'
  | 'verificada'
  | 'esperando_cuota'
  | 'bloqueada'
  | 'descartada'
  | 'reabierta'

export type TaskPriority = 'p0' | 'p1' | 'p2' | 'p3'
export type TaskOrigin = 'insights' | 'shield' | 'manual'

export type TaskActivityKind =
  | 'created'
  | 'state_change'
  | 'decision'
  | 'agent_run'
  | 'pr_link'
  | 'comment'
  | 'assignment'
  | 'priority_change'
  | 'updated'

export type TaskLinkKind = 'pullreq' | 'ai_task' | 'insights_issue' | 'shield_finding'

export interface TypesTask {
  repo_id?: number
  number?: number
  title?: string
  description?: string
  origin?: TaskOrigin
  origin_ref?: string
  priority?: TaskPriority
  state?: TaskState
  state_reason?: string
  parent_id?: number
  due?: number
  created_by?: number
  created?: number
  updated?: number
  resolved?: number
  verified?: number
  assignee_principal_id?: number
  assignee_group_id?: number
  assignee_is_agent?: boolean
  // Only set by the aggregated (space level) listings.
  repo_path?: string
}

export interface TypesTaskActivity {
  id?: number
  task_id?: number
  kind?: TaskActivityKind
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  payload?: any
  principal_id?: number
  created?: number
}

export interface TypesTaskLink {
  task_id?: number
  kind?: TaskLinkKind
  ref_id?: string
  created?: number
}

export interface TaskAssigneeInput {
  principal_id?: number
  group_id?: number
  is_agent?: boolean
}

export interface TaskCreateInput {
  title: string
  description?: string
  priority?: TaskPriority
  due?: number
  assignee?: TaskAssigneeInput
}

export interface TaskUpdateInput {
  title?: string
  description?: string
  priority?: TaskPriority
  due?: number
  clear_due?: boolean
  assignee?: TaskAssigneeInput
}

export interface TaskStateInput {
  state: TaskState
  reason?: string
}

export const TASK_STATES: TaskState[] = [
  'abierta',
  'triage',
  'en_curso',
  'en_revision',
  'resuelta',
  'verificada',
  'esperando_cuota',
  'bloqueada',
  'descartada',
  'reabierta'
]

// Columns of the kanban board.
export const TASK_MAIN_FLOW_STATES: TaskState[] = [
  'abierta',
  'triage',
  'en_curso',
  'en_revision',
  'resuelta',
  'verificada'
]
export const TASK_SIDE_STATES: TaskState[] = ['esperando_cuota', 'bloqueada', 'descartada', 'reabierta']

export const TASK_PRIORITIES: TaskPriority[] = ['p0', 'p1', 'p2', 'p3']
export const TASK_ORIGINS: TaskOrigin[] = ['insights', 'shield', 'manual']

// Mirror of the transition table of the backend (app/services/task/state.go). The backend stays the
// source of truth: this is only used to offer sensible buttons, errors are always shown to the user.
const SIDE_ENTRY: TaskState[] = ['esperando_cuota', 'bloqueada']
const RESUME_TARGETS: TaskState[] = ['abierta', 'en_curso', 'en_revision', 'reabierta']

export const TASK_TRANSITIONS: Record<TaskState, TaskState[]> = {
  abierta: ['triage', 'en_curso', ...SIDE_ENTRY, 'descartada'],
  triage: ['en_curso', ...SIDE_ENTRY, 'descartada'],
  en_curso: ['triage', 'en_revision', ...SIDE_ENTRY, 'descartada'],
  en_revision: ['en_curso', 'resuelta', ...SIDE_ENTRY, 'descartada'],
  resuelta: ['verificada', 'reabierta', 'descartada'],
  verificada: ['reabierta'],
  reabierta: ['triage', 'en_curso', ...SIDE_ENTRY, 'descartada'],
  esperando_cuota: ['triage', 'descartada', ...RESUME_TARGETS],
  bloqueada: ['triage', 'descartada', ...RESUME_TARGETS],
  descartada: []
}

export const getAllowedTransitions = (state?: TaskState): TaskState[] => (state ? TASK_TRANSITIONS[state] || [] : [])

export const isSideState = (state?: TaskState): boolean => state === 'esperando_cuota' || state === 'bloqueada'

export const TASK_STATE_COLORS: Record<TaskState, { fg: string; bg: string }> = {
  abierta: { fg: 'var(--green-700)', bg: 'var(--green-50)' },
  triage: { fg: 'var(--orange-700)', bg: 'var(--orange-50)' },
  en_curso: { fg: 'var(--primary-7)', bg: 'var(--primary-1)' },
  en_revision: { fg: 'var(--purple-700)', bg: 'var(--purple-50)' },
  resuelta: { fg: 'var(--teal-700)', bg: 'var(--teal-50)' },
  verificada: { fg: 'var(--green-800)', bg: 'var(--green-100)' },
  esperando_cuota: { fg: 'var(--yellow-800)', bg: 'var(--yellow-50)' },
  bloqueada: { fg: 'var(--red-700)', bg: 'var(--red-50)' },
  descartada: { fg: 'var(--grey-600)', bg: 'var(--grey-100)' },
  reabierta: { fg: 'var(--orange-800)', bg: 'var(--orange-100)' }
}

export const getTaskApiPath = (repoPath?: string): string => `/api/v1/repos/${repoPath}/+/tasks`

export const formatConfidence = (confidence?: number): string =>
  typeof confidence === 'number' ? confidence.toFixed(2) : '-'

export const getAssigneeLabel = (
  task: Pick<TypesTask, 'assignee_is_agent' | 'assignee_principal_id' | 'assignee_group_id'>,
  getString: (key: StringKeys, vars?: Record<string, unknown>) => string
): string => {
  if (task.assignee_is_agent) {
    return getString('tasks.agent')
  }
  if (task.assignee_principal_id) {
    return getString('tasks.principal', { id: task.assignee_principal_id })
  }
  if (task.assignee_group_id) {
    return getString('tasks.group', { id: task.assignee_group_id })
  }
  return getString('tasks.unassigned')
}

// Converts between epoch milliseconds and the value of an <input type="date">.
export const dueToInputValue = (due?: number): string => (due ? new Date(due).toISOString().slice(0, 10) : '')
export const inputValueToDue = (value: string): number | undefined => {
  const time = Date.parse(`${value}T00:00:00Z`)
  return Number.isNaN(time) ? undefined : time
}

// Maps the origin_ref of the tasks of an origin to their number. Insights issues and Shield findings only
// expose the internal id of their task, so the number shown to users is resolved through the task list.
export const useTaskNumbersByOriginRef = (repoPath: string | undefined, origin: TaskOrigin): Record<string, number> => {
  const { data } = useGet<TypesTask[]>({
    path: getTaskApiPath(repoPath),
    queryParams: { origin, limit: 100 },
    lazy: !repoPath
  })
  const result: Record<string, number> = {}
  data?.forEach(task => {
    if (task.origin_ref && task.number) {
      result[task.origin_ref] = task.number
    }
  })
  return result
}
