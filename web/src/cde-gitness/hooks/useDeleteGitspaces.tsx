import { useMutate } from 'restful-react'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useDeleteGitspace } from 'services/cde'

export const useDeleteGitspaces = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useMutate<any>({
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
