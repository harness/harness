import { useAppContext } from 'AppContext'
import { ListInfraProvidersQueryParams, useListInfraProviders } from 'services/cde'

export const useInfraListingApi = ({ queryParams }: { queryParams: ListInfraProvidersQueryParams }) => {
  const { accountInfo } = useAppContext()

  const infraListing = useListInfraProviders({
    accountIdentifier: accountInfo?.identifier,
    queryParams,
    lazy: true
  })

  return infraListing
}
