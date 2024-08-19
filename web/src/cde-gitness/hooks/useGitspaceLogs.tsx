import { useGet } from 'restful-react'
import { useEffect } from 'react'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useGetGitspaceInstanceLogs } from 'services/cde'

export const useGitspacesLogs = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useGet<any>({
    path: `api/v1/gitspaces/${space}/${gitspaceId}/+/logs/stream`,
    debounce: 500,
    lazy: true
  })

  const cde = useGetGitspaceInstanceLogs({
    projectIdentifier,
    orgIdentifier,
    accountIdentifier,
    gitspace_identifier: gitspaceId,
    lazy: true
  })

  useEffect(() => {
    if (!standalone) {
      cde.refetch()
    }
  }, [])

  return standalone ? gitness : cde
}
