import { useGet } from 'restful-react'
import { useEffect } from 'react'
import type { TypesGitspaceConfig } from 'cde-gitness/services'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useFindGitspace } from 'services/cde'

export const useGitspaceDetails = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useGet<TypesGitspaceConfig>({
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+`,
    debounce: 500,
    lazy: true
  })

  const cde = useFindGitspace({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspace_identifier: gitspaceId || '',
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
