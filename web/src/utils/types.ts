import type { DiffFile } from 'diff2html/lib/types'

export interface DiffFileEntry extends DiffFile {
  containerId: string
  contentId: string
}
