export enum GitspaceEventType {
  GitspaceActionStopCompleted = 'gitspaceActionStopCompleted',
  GitspaceActionStopFailed = 'gitspaceActionStopFailed',
  GitspaceActionStartCompleted = 'gitspaceActionStartCompleted',
  GitspaceActionStartFailed = 'gitspaceActionStartFailed',
  InfraUnprovisioningCompleted = 'infraUnprovisioningCompleted',
  AgentGitspaceCreationStart = 'agentGitspaceCreationStart'
}

export const pollEventsList = [
  GitspaceEventType.GitspaceActionStopCompleted,
  GitspaceEventType.GitspaceActionStopFailed,
  GitspaceEventType.GitspaceActionStartCompleted,
  GitspaceEventType.GitspaceActionStartFailed,
  GitspaceEventType.InfraUnprovisioningCompleted
]
