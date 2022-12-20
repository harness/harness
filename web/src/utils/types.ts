import type { DiffFile } from 'diff2html/lib/types'

export interface PullRequestPayload {
  // TODO: Use from service when it's ready
  sourceBranch: string
  targetBranch: string
  sourceRepoRef?: string
  title: string
  description?: string
}

export interface PullRequestResponse {
  // TODO: Use from service when it's ready
  id: number
  createdBy: number
  created: number
  updated: number
  number: number
  state: string
  title: string
  description: string
  sourceRepoID: number
  sourceBranch: string
  targetRepoID: number
  targetBranch: string
  mergedBy: Unknown
  merged: Unknown
  merge_strategy: Unknown
}

export interface DiffFileEntry extends DiffFile {
  containerId: string
  contentId: string
}
