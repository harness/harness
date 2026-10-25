/*
 * Copyright 2026 Harness, Inc.
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
import type { TaskPriority, TaskState } from 'utils/TaskUtils'

// Types and hooks for the AI agent panel REST API (/api/v1/spaces/{space_ref}/ai/agent-panel).
// The generated services do not include it yet, hence the manual types.

export interface AIAgentQuota {
  day?: string
  limit: number
  used: number
  used_by_class: Record<string, number>
  remaining_by_class: Record<string, number>
  space_used_today: number
}

export interface AIAgentTask {
  number: number
  repo_path: string
  title: string
  state: TaskState
  priority: TaskPriority
  updated: number
  model?: string
}

export interface AIAgentWaitingQuota {
  count: number
  tasks: AIAgentTask[]
}

export interface AIAgentKindSuccess {
  kind: string
  rate: number
  samples: number
}

export interface AIAgentModelSuccess {
  model: string
  kinds: AIAgentKindSuccess[]
}

export interface AIAgentRun {
  task_number: number
  repo_path: string
  title: string
  model: string
  outcome: string
  requests_used: number
  created: number
}

export interface AIAgentPanel {
  enabled: boolean
  quota: AIAgentQuota
  active_tasks: AIAgentTask[]
  waiting_quota: AIAgentWaitingQuota
  model_success: AIAgentModelSuccess[]
  latest_runs: AIAgentRun[]
}

export const getAIAgentPanelPath = (space?: string): string => `/api/v1/spaces/${space}/+/ai/agent-panel`

export const useGetAIAgentPanel = (space?: string) =>
  useGet<AIAgentPanel>({
    path: getAIAgentPanelPath(space),
    lazy: !space
  })

export const getQuotaPercent = (quota?: Pick<AIAgentQuota, 'limit' | 'used'>): number =>
  quota && quota.limit > 0 ? Math.min(1, Math.max(0, quota.used / quota.limit)) : 0

export const formatSuccessRate = (rate: number): string => `${Math.round(rate * 100)}%`
