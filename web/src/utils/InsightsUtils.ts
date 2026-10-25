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

import type { BadgeColors } from './ShieldUtils'

// Types and helpers for the Insights REST API (/api/v1/repos/{repo_ref}/insights).
// The generated services do not include Insights yet, hence the manual types.

export type InsightsIssueStatus = 'unresolved' | 'resolved' | 'ignored' | 'regressed'
export type InsightsIssueSort = 'last_seen' | 'times_seen'

export interface TypesInsightsProject {
  id?: number
  repo_id?: number
  identifier?: string
  public_key?: string
  enabled?: boolean
  rate_limit_per_minute?: number
  sample_rate?: number
  created?: number
  updated?: number
  dsn?: string
}

export interface InsightsProjectCreateInput {
  identifier: string
  rate_limit_per_minute?: number
  sample_rate?: number
}

export interface InsightsProjectUpdateInput {
  enabled?: boolean
  rate_limit_per_minute?: number
  sample_rate?: number
}

export interface TypesInsightsIssue {
  id?: number
  project_id?: number
  repo_id?: number
  fingerprint?: string
  title?: string
  culprit?: string
  level?: string
  platform?: string
  first_seen?: number
  last_seen?: number
  times_seen?: number
  users_seen?: number
  first_release?: string
  last_release?: string
  status?: InsightsIssueStatus
  resolved_in_release?: string
  task_id?: number
  created?: number
  updated?: number
}

export interface InsightsIssueUpdateInput {
  status: InsightsIssueStatus
  resolved_in_release?: string
}

export interface TypesInsightsEvent {
  id?: number
  issue_id?: number
  project_id?: number
  event_id?: string
  timestamp?: number
  release?: string
  environment?: string
  level?: string
  message?: string
  // Sanitized event as a JSON string.
  payload?: string
  created?: number
}

export interface InsightsFrame {
  filename: string
  lineno?: number
  function?: string
  inApp?: boolean
}

export interface InsightsException {
  type: string
  value: string
  frames: InsightsFrame[]
}

export const INSIGHTS_ISSUE_STATUSES: InsightsIssueStatus[] = ['unresolved', 'regressed', 'resolved', 'ignored']
export const INSIGHTS_LEVELS = ['fatal', 'error', 'warning', 'info', 'debug']

export const getInsightsApiPath = (repoPath?: string): string => `/api/v1/repos/${repoPath}/+/insights`

export const INSIGHTS_STATUS_COLORS: Record<InsightsIssueStatus, BadgeColors> = {
  unresolved: { fg: 'var(--red-700)', bg: 'var(--red-50)' },
  regressed: { fg: 'var(--orange-800)', bg: 'var(--orange-100)' },
  resolved: { fg: 'var(--green-700)', bg: 'var(--green-50)' },
  ignored: { fg: 'var(--grey-600)', bg: 'var(--grey-100)' }
}

export const getLevelColors = (level?: string): BadgeColors => {
  switch (level) {
    case 'fatal':
      return { fg: 'var(--red-800)', bg: 'var(--red-100)' }
    case 'error':
      return { fg: 'var(--red-700)', bg: 'var(--red-50)' }
    case 'warning':
      return { fg: 'var(--yellow-800)', bg: 'var(--yellow-50)' }
    case 'info':
      return { fg: 'var(--primary-7)', bg: 'var(--primary-1)' }
    default:
      return { fg: 'var(--grey-600)', bg: 'var(--grey-100)' }
  }
}

const asArray = (value: unknown): unknown[] => (Array.isArray(value) ? value : [])
const asString = (value: unknown): string => (typeof value === 'string' ? value : '')

// Extracts the exceptions (type, value and frames) from the sanitized event payload. Both the
// `{ exception: { values: [...] } }` and the `{ exception: [...] }` shapes are accepted. Sentry stores frames
// outermost first, they are returned innermost first.
export const parseEventExceptions = (payload?: string): InsightsException[] => {
  if (!payload) {
    return []
  }
  try {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const event = JSON.parse(payload) as any
    const exception = event?.exception
    const values = asArray(Array.isArray(exception) ? exception : exception?.values)
    return values.map(raw => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const v = (raw || {}) as any
      const frames = asArray(v.stacktrace?.frames)
        .map(f => {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const frame = (f || {}) as any
          return {
            filename: asString(frame.filename) || asString(frame.abs_path) || asString(frame.module),
            lineno: typeof frame.lineno === 'number' ? frame.lineno : undefined,
            function: asString(frame.function),
            inApp: !!frame.in_app
          } as InsightsFrame
        })
        .reverse()
      return { type: asString(v.type), value: asString(v.value), frames }
    })
  } catch {
    return []
  }
}
