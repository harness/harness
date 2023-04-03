import type { DiffFile } from 'diff2html/lib/types'
import type { TypesPullReqActivity } from 'services/code'

export interface DiffFileEntry extends DiffFile {
  fileId: string
  fileTitle: string
  containerId: string
  contentId: string
  fileActivities?: TypesPullReqActivity[]
  activities?: TypesPullReqActivity[]
}

export interface GitBlameEntry {
  Commit: {
    SHA: string
    Title: string
    Message: string
    Author: {
      Identity: {
        Name: string
        Email: string
      }
      When: string
    }
    Committer: {
      Identity: {
        Name: string
        Email: string
      }
      When: string
    }
  }
  Lines: string[]
}

export type GitBlameResponse = GitBlameEntry[]
