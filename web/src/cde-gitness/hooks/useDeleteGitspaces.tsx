import { useMutate } from 'restful-react'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { DeleteGitspacePathParams, useDeleteGitspace, UsererrorError } from 'services/cde'

export const useDeleteGitspaces = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useMutate<void, UsererrorError, void, string, DeleteGitspacePathParams>({
    verb: 'DELETE',
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+`
  })

  const cde = useDeleteGitspace({
    projectIdentifier,
    orgIdentifier,
    accountIdentifier
  })

  return standalone ? gitness : cde
}
