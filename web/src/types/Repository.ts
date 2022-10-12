export interface Repository {
  id: number
  spaceId: number
  pathName: string
  path: string
  name: string
  description?: string
  isPublic: boolean
  createdBy: number
  created: number
  updated: number
  forkId: number
  numForks: number
  numPulls: number
  numClosedPulls: number
  numOpenPulls: number
}
