export interface KeywordSearchResponse {
  file_matches: FileMatch[]
  stats: Stats
}

export interface FileMatch {
  file_name: string
  repo_path: string
  repo_branch: string
  language: string
  matches: Match[]
}

export interface Match {
  line_num: number
  fragments: Fragment[]
  before: string
  after: string
}

export interface Fragment {
  pre: string
  match: string
  post: string
}

export interface Stats {
  total_files: number
  total_matches: number
}
