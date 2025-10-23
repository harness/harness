import { useEffect, useState } from 'react'
import { HYBRID_VM_GCP, regionType } from 'cde-gitness/constants'
import type { TypesInfraProviderConfig } from 'services/cde'

interface UseGcpInfrastructureProps {
  listResponse?: TypesInfraProviderConfig[] | null
  loading?: boolean
  refetch?: () => void
}

export const useGcpInfrastructure = ({ listResponse, loading, refetch }: UseGcpInfrastructureProps) => {
  const [gcpInfraDetails, setGcpInfraDetails] = useState<TypesInfraProviderConfig>()
  const [gcpRegionData, setGcpRegionData] = useState<regionType[]>()
  const [isConnected, setIsConnected] = useState(true)

  useEffect(() => {
    const gcpInfra =
      listResponse?.find((infraProvider: TypesInfraProviderConfig) => infraProvider?.type === HYBRID_VM_GCP) || null

    // Process GCP infrastructure
    if (gcpInfra) {
      const regions: regionType[] = []
      const region_configs = gcpInfra?.metadata?.region_configs ?? {}
      Object.keys(region_configs)?.forEach((key: string) => {
        const { region_name, proxy_subnet_ip_range, default_subnet_ip_range, certificates } = region_configs?.[key]
        regions.push({
          region_name,
          proxy_subnet_ip_range,
          default_subnet_ip_range,
          dns: certificates?.contents?.[0]?.dns_managed_zone_name,
          domain: certificates?.contents?.[0]?.domain,
          machines: gcpInfra?.resources?.filter((item: any) => item?.metadata?.region_name === region_name) ?? []
        })
      })
      setIsConnected(true)
      setGcpInfraDetails(gcpInfra)
      setGcpRegionData(regions)
    } else {
      setGcpInfraDetails(undefined)
      setGcpRegionData([])
    }
  }, [listResponse])

  return {
    gcpInfraDetails,
    gcpRegionData,
    setGcpRegionData,
    isConnected,
    loading,
    refetch
  }
}
