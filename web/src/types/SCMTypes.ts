//
// TODO: Types should be auto-generated from backend
//
export interface RepositoryDTO {
  created: number
  createdBy: number
  defaultBranch: string
  description: string
  forkId: number
  id: number
  isPublic: boolean
  name: string
  numClosedPulls: number
  numForks: number
  numOpenPulls: number
  numPulls: number
  path: string
  pathName: string
  spaceId: number
  updated: number
}

export interface CreateRepositoryBody {
  defaultBranch: string
  description?: string
  forkId?: number
  gitIgnore: string
  isPublic: boolean
  license: string
  name: string
  pathName: string
  readme: boolean
  spaceId: number | string
}

export enum GitContentType {
  FILE = 'file',
  DIR = 'dir',
  SYMLINK = 'symlink',
  SUBMODULE = 'submodule'
}
