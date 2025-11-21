/**
 * SuggestionBlock represents a suggestion block.
 */
export interface SuggestionBlock {
  /** Orginal source - lines that have suggestion comment */
  source: string

  /** Language of the diff/file */
  lang?: string

  /** Comment id */
  commentId?: number

  /** Applied check sum */
  appliedCheckSum?: string

  /** Applied commit SHA */
  appliedCommitSha?: string
}
