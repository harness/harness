import type { DiffFile } from 'diff2html/lib/types'

export interface DiffFileEntry extends DiffFile {
  containerId: string
  contentId: string
}

// TODO: Use proper type when API supports it
export interface CommentThreadEntry {
  id: string
  author: string
  created: string
  updated: string
  content: string
  emoji?: EmojiInfo[]
}

// TODO: Use proper type when API supports it
export interface UserProfile {
  id: string
  name: string
  email?: string
}

// TODO: Use proper type when API supports it
export interface EmojiInfo {
  name: string
  by: UserProfile[]
}
