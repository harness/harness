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

// Types and helpers for the Shield REST API (/api/v1/repos/{repo_ref}/shield).
// The generated services do not include Shield yet, hence the manual types.

export type ShieldFindingKind = 'dependency' | 'sast' | 'secret' | 'image' | 'iac'
export type ShieldFindingState = 'open' | 'fixed' | 'ignored' | 'accepted_risk'
export type ShieldScanStatus = 'pending' | 'running' | 'done' | 'failed'
export type ShieldScanTrigger = 'push' | 'pullreq' | 'schedule' | 'manual'

export interface TypesShieldScan {
  id?: number
  repo_id?: number
  commit_sha?: string
  ref?: string
  trigger?: ShieldScanTrigger
  scanner?: string
  status?: ShieldScanStatus
  error?: string
  packages_count?: number
  findings_count?: number
  started?: number
  finished?: number
  created?: number
}

export interface TypesShieldFinding {
  id?: number
  repo_id?: number
  fingerprint?: string
  scanner?: string
  kind?: ShieldFindingKind
  ecosystem?: string
  package?: string
  installed_version?: string
  manifest_path?: string
  vuln_id?: string
  aliases?: string[]
  summary?: string
  cvss_score?: number
  epss?: number
  in_kev?: boolean
  reachable?: 'unknown' | 'reachable' | 'not_reachable'
  risk_score?: number
  fix_strategy?: string
  min_fixed_version?: string
  fixed_versions?: string[]
  state?: ShieldFindingState
  state_reason?: string
  first_seen?: number
  last_seen?: number
  fixed_at?: number
  task_id?: number
  created?: number
  updated?: number
}

export interface ShieldFindingUpdateInput {
  state: 'ignored' | 'accepted_risk'
  reason: string
}

export type BadgeColors = { fg: string; bg: string }

export const SHIELD_FINDING_KINDS: ShieldFindingKind[] = ['dependency', 'sast', 'secret', 'image', 'iac']
export const SHIELD_FINDING_STATES: ShieldFindingState[] = ['open', 'fixed', 'ignored', 'accepted_risk']
export const SHIELD_MIN_RISK_OPTIONS = [25, 50, 75]

export const getShieldApiPath = (repoPath?: string): string => `/api/v1/repos/${repoPath}/+/shield`

export const SHIELD_STATE_COLORS: Record<ShieldFindingState, BadgeColors> = {
  open: { fg: 'var(--red-700)', bg: 'var(--red-50)' },
  fixed: { fg: 'var(--green-700)', bg: 'var(--green-50)' },
  ignored: { fg: 'var(--grey-600)', bg: 'var(--grey-100)' },
  accepted_risk: { fg: 'var(--orange-700)', bg: 'var(--orange-50)' }
}

export const SHIELD_SCAN_STATUS_COLORS: Record<ShieldScanStatus, BadgeColors> = {
  pending: { fg: 'var(--grey-600)', bg: 'var(--grey-100)' },
  running: { fg: 'var(--primary-7)', bg: 'var(--primary-1)' },
  done: { fg: 'var(--green-700)', bg: 'var(--green-50)' },
  failed: { fg: 'var(--red-700)', bg: 'var(--red-50)' }
}

// Risk score is in [0, 100].
export const getRiskColors = (risk = 0): BadgeColors => {
  if (risk >= 75) {
    return { fg: 'var(--red-800)', bg: 'var(--red-100)' }
  }
  if (risk >= 50) {
    return { fg: 'var(--orange-800)', bg: 'var(--orange-100)' }
  }
  if (risk >= 25) {
    return { fg: 'var(--yellow-800)', bg: 'var(--yellow-50)' }
  }
  return { fg: 'var(--grey-700)', bg: 'var(--grey-100)' }
}
