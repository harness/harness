import { useEffect, useState } from 'react'
import { HYBRID_VM_AWS, regionType } from 'cde-gitness/constants'
import type { TypesInfraProviderConfig } from 'services/cde'

interface UseAwsInfrastructureProps {
  listResponse?: TypesInfraProviderConfig[] | null
  loading?: boolean
  refetch?: () => void
}

export const useAwsInfrastructure = ({ listResponse, loading, refetch }: UseAwsInfrastructureProps) => {
  const [awsInfraDetails, setAwsInfraDetails] = useState<TypesInfraProviderConfig>()
  const [awsRegionData, setAwsRegionData] = useState<regionType[]>()
  const [isConnected, setIsConnected] = useState(true)

  useEffect(() => {
    const awsInfra =
      listResponse?.find((infraProvider: TypesInfraProviderConfig) => infraProvider?.type === HYBRID_VM_AWS) || null

    if (awsInfra) {
      const regions: regionType[] = []
      const region_configs = awsInfra?.metadata?.region_configs ?? {}
      Object.keys(region_configs)?.forEach((key: string) => {
        const { region_name, proxy_subnet_ip_range, default_subnet_ip_range, certificates } = region_configs?.[key]
        regions.push({
          region_name,
          proxy_subnet_ip_range,
          default_subnet_ip_range,
          dns: certificates?.contents?.[0]?.dns_managed_zone_name,
          domain: certificates?.contents?.[0]?.domain,
          machines: awsInfra?.resources?.filter((item: any) => item?.metadata?.region_name === region_name) ?? []
        })
      })
      setIsConnected(true)
      setAwsInfraDetails(awsInfra)
      setAwsRegionData(regions)
    } else {
      setAwsInfraDetails(undefined)
      setAwsRegionData([])
    }
  }, [listResponse])

  return {
    awsInfraDetails,
    awsRegionData,
    setAwsRegionData,
    isConnected,
    loading,
    refetch
  }
}
