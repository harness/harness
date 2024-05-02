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

import type { DiffFile } from 'diff2html/lib/types'

export interface DiffFileEntry extends DiffFile {
  fileId: string
  filePath: string
  containerId: string
  contentId: string
  fileViews?: Map<string, string>
}

export type FCWithChildren<T> = React.FC<React.PropsWithChildren<T>>

export enum AidaClient {
  CODE_SEMANTIC_SEARCH = 'CODE_SEMANTIC_SEARCH',
  CODE_PR_SUMMARY = 'CODE_PR_SUMMARY'
}
export interface TelemeteryProps {
  aidaClient: AidaClient
  metadata?: Record<string, string | boolean | number>
}

export interface UsefulOrNotProps {
  telemetry: TelemeteryProps
  allowFeedback?: boolean
  allowCreateTicket?: boolean
  onVote?: (vote: Vote) => void
  className?: string
}

enum Vote {
  None,
  Up,
  Down
}

export interface Identifier {
  accountId: string
  orgIdentifier: string
  projectIdentifier: string
}
