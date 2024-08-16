import { useMutate } from 'restful-react'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useGitspaceAction } from 'services/cde'

export const useGitspaceActions = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useMutate({
    verb: 'POST',
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+/actions`
  })

  const cde = useGitspaceAction({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspace_identifier: gitspaceId
  })

  return standalone ? gitness : cde
}
