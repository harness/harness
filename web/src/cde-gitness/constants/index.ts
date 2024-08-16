export enum StandaloneIDEType {
  VSCODE = 'vs_code',
  VSCODEWEB = 'vs_code_web'
}

export enum EnumGitspaceCodeRepoType {
  GITHUB = 'github',
  GITLAB = 'gitlab',
  HARNESS_CODE = 'harness_code',
  BITBUCKET = 'bitbucket',
  UNKNOWN = 'unknown',
  GITNESS = 'gitness'
}

export enum GitspaceStatus {
  RUNNING = 'running',
  STOPPED = 'stopped',
  UNKNOWN = 'unknown',
  ERROR = 'error',
  STARTING = 'starting',
  STOPPING = 'stopping',
  UNINITIALIZED = 'uninitialized'
}

export enum GitspaceActionType {
  START = 'start',
  STOP = 'stop'
}

export enum GitspaceRegion {
  USEast = 'us-east',
  USWest = 'us-west',
  Europe = 'Europe',
  Australia = 'Australia'
}
