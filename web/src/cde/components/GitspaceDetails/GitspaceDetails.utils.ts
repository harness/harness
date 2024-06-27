import { isUndefined } from 'lodash-es'
import type { UseStringsReturn } from 'framework/strings'
import { GitspaceActionType, GitspaceStatus } from 'cde/constants'
import type { EnumGitspaceStateType, OpenapiGetGitspaceEventResponse } from 'services/cde'

const getTitleWhenNotPolling = ({
  state,
  getString
}: {
  state?: EnumGitspaceStateType
  getString: UseStringsReturn['getString']
}) => {
  if (state === GitspaceStatus.STOPPED) {
    return getString('cde.details.gitspaceStopped')
  } else if (state === GitspaceStatus.RUNNING) {
    return getString('cde.details.gitspaceRunning')
  } else if (isUndefined(state)) {
    getString('cde.details.noData')
  }
}

export const getGitspaceStatusLabel = ({
  state,
  eventData,
  isPolling,
  startPolling,
  getString,
  mutateLoading
}: {
  state?: EnumGitspaceStateType
  eventData: OpenapiGetGitspaceEventResponse[] | null
  isPolling: boolean
  getString: UseStringsReturn['getString']
  startPolling?: GitspaceActionType
  mutateLoading?: boolean
}) => {
  if (mutateLoading || Boolean(startPolling) || isPolling) {
    if (isPolling && eventData?.[eventData?.length - 1]?.message) {
      return eventData?.[eventData?.length - 1]?.message
    } else if (startPolling === GitspaceActionType.START) {
      return getString('cde.startingGitspace')
    } else if (startPolling === GitspaceActionType.STOP) {
      return getString('cde.stopingGitspace')
    } else {
      return getString('cde.details.fetchingGitspace')
    }
  } else {
    return getTitleWhenNotPolling({ getString, state })
  }
}
