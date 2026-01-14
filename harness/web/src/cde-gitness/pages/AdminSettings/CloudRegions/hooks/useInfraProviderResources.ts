import { useMemo } from 'react'
import { INFRA_PROVIDER_DISPLAY_NAMES } from 'cde-gitness/constants'
import type { TypesGitspaceSettingsResponse } from 'services/cde'
import type { InfraProviderResource, RegionData } from 'cde-gitness/utils/cloudRegionsUtils'

/**
 * Custom hook to process infrastructure provider resources from API data
 * Transforms raw API data into display-friendly format with proper names
 */
export const useInfraProviderResources = (settings: TypesGitspaceSettingsResponse | null): InfraProviderResource[] => {
  return useMemo(() => {
    const infraResources = settings?.available_settings?.infra_provider_resources
    if (!infraResources) return []

    return Object.entries(infraResources).map(([key, regions]) => ({
      name: INFRA_PROVIDER_DISPLAY_NAMES[key] || key,
      key,
      regions: regions as RegionData[]
    }))
  }, [settings])
}
