import type { TypesDevcontainerImage, TypesGitspaceSettingsResponse } from 'services/cde'

export interface MachineType {
  identifier: string
  name: string
  cpu: string
  memory: string
  disk: string
  network: string
  region: string
  metadata: Record<string, any>
  infra_provider_type: string
}

export interface RegionData {
  region: string
  region_display_name: string
  machine_types: MachineType[]
}

export interface InfraProviderResource {
  name: string
  key: string
  regions: RegionData[]
}

export interface AdminSettingsFormValues {
  cloudRegions: {
    [infraProvider: string]: {
      [region: string]: {
        [machineTypeId: string]: boolean
      }
    }
  }
  gitspaceImages?: TypesDevcontainerImage
}

export const getCloudRegionFieldPath = (infraProvider: string, region: string, machineTypeId: string): string => {
  return `cloudRegions.${infraProvider}.${region}.${machineTypeId}`
}

export const processInfraProviderDenyList = (
  settings: TypesGitspaceSettingsResponse,
  infraProviderResources: InfraProviderResource[]
): { [infraProvider: string]: { [region: string]: { [machineTypeId: string]: boolean } } } => {
  const newCloudRegionsValues: {
    [infraProvider: string]: { [region: string]: { [machineTypeId: string]: boolean } }
  } = {}

  infraProviderResources.forEach(provider => {
    const providerKey = provider.key
    const infraProviderSettings = settings.settings?.infra_provider?.[providerKey]
    const deniedList = infraProviderSettings?.access_list?.list || []
    const deniedSet = new Set(deniedList)

    newCloudRegionsValues[providerKey] = {}

    provider.regions.forEach((regionData: RegionData) => {
      newCloudRegionsValues[providerKey][regionData.region] = {}

      regionData.machine_types.forEach((machineType: MachineType) => {
        newCloudRegionsValues[providerKey][regionData.region][machineType.identifier] = !deniedSet.has(
          machineType.identifier
        )
      })
    })
  })

  return newCloudRegionsValues
}
