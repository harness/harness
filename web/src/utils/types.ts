import type { DiffFile } from 'diff2html/lib/types'
import type { TypesPullReqActivity } from 'services/code'

export interface DiffFileEntry extends DiffFile {
  fileId: string
  filePath: string
  containerId: string
  contentId: string
  fileActivities?: TypesPullReqActivity[]
  activities?: TypesPullReqActivity[]
  fileViews?: Map<string, string>
}
