import { useAppContext } from 'AppContext'
import { useListInfraProviders } from 'services/cde'

export const useInfraListingApi = () => {
  const { accountInfo } = useAppContext()

  const infraListing = useListInfraProviders({
    accountIdentifier: accountInfo?.identifier,
    lazy: true
  })

  return infraListing
}
