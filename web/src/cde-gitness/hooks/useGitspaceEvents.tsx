import { useGet } from 'restful-react'
import { useEffect } from 'react'
import type { TypesGitspaceEventResponse } from 'cde-gitness/services'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useGetGitspaceEvents } from 'services/cde'

export const useGitspaceEvents = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useGet<TypesGitspaceEventResponse[]>({
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+/events`,
    debounce: 500,
    lazy: true
  })

  const cde = useGetGitspaceEvents({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspace_identifier: gitspaceId,
    lazy: true
  })

  useEffect(() => {
    if (standalone) {
      gitness.refetch()
    } else {
      cde.refetch()
    }
  }, [])

  return standalone ? gitness : cde
}
