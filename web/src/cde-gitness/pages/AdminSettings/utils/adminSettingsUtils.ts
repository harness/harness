import type {
  TypesGitspaceSettingsData,
  TypesGitspaceSettingsResponse,
  TypesGitspaceRegionMachines,
  TypesInfraProviderSettings
} from 'services/cde'
import { scmOptionsCDE } from 'cde-gitness/pages/GitspaceCreate/CDECreateGitspace'
import type { IDEOption } from 'cde-gitness/constants'
import type { EnumInfraProviderType } from 'cde-gitness/services'

export interface AdminSettingsFormValues {
  gitProviders: {
    [key: string]: boolean
  }
  codeEditors: {
    [key: string]: boolean
  }
  cloudRegions: {
    [infraProvider: string]: {
      [region: string]: {
        [machineTypeId: string]: boolean
      }
    }
  }
}

/**
 * Creates initial form values for all admin settings tabs
 */
export const createInitialValues = (availableEditors: IDEOption[]): AdminSettingsFormValues => {
  return {
    gitProviders: scmOptionsCDE.reduce((acc, provider) => {
      acc[provider.value] = true
      return acc
    }, {} as { [key: string]: boolean }),
    codeEditors: availableEditors.reduce((acc, editor) => {
      acc[editor.value] = true
      return acc
    }, {} as { [key: string]: boolean }),
    cloudRegions: {}
  }
}

/**
 * Transforms Git Providers form data to API payload format
 */
export const transformGitProvidersToPayload = (formValues: AdminSettingsFormValues) => {
  const allProviders = scmOptionsCDE.map(p => p.value)
  const deniedProviders = allProviders.filter(provider => !formValues.gitProviders[provider])

  return {
    access_list: {
      mode: 'deny' as const,
      list: deniedProviders
    }
  }
}

/**
 * Transforms Code Editors form data to API payload format
 */
export const transformCodeEditorsToPayload = (formValues: AdminSettingsFormValues, availableEditors: IDEOption[]) => {
  const allEditors = availableEditors.map(editor => editor.value)
  const deniedEditors = allEditors.filter(editor => !formValues.codeEditors[editor])

  return {
    access_list: {
      mode: 'deny' as const,
      list: deniedEditors
    }
  }
}

/**
 * Transforms Cloud Regions form data to API payload format
 */
export const transformCloudRegionsToPayload = (
  formValues: AdminSettingsFormValues,
  settings: TypesGitspaceSettingsResponse | null
) => {
  const infraProviderPayload: { [key: string]: TypesInfraProviderSettings } = {}
  const availableInfraProviders = settings?.available_settings?.infra_provider_resources || {}

  Object.keys(availableInfraProviders).forEach(providerKey => {
    const providerRegions = availableInfraProviders[providerKey] || []
    const deniedMachineTypes: string[] = []

    // Collect denied machine types for this provider
    providerRegions.forEach((regionData: TypesGitspaceRegionMachines) => {
      regionData.machine_types?.forEach(machineType => {
        const regionName = regionData.region || ''
        const isEnabled = formValues.cloudRegions?.[providerKey]?.[regionName]?.[machineType?.identifier || '']
        if (isEnabled === false && machineType?.identifier) {
          deniedMachineTypes.push(machineType.identifier)
        }
      })
    })

    // Construct provider payload with required structure
    infraProviderPayload[providerKey] = {
      access_list: {
        mode: 'deny' as const,
        list: deniedMachineTypes
      },
      infra_provider_type: providerKey as EnumInfraProviderType
    }
  })

  return infraProviderPayload
}

/**
 * Constructs the complete API payload for admin settings
 */
export const buildAdminSettingsPayload = (
  formValues: AdminSettingsFormValues,
  availableEditors: IDEOption[],
  settings: TypesGitspaceSettingsResponse | null
): TypesGitspaceSettingsData => {
  return {
    ...settings?.settings,
    gitspace_config: {
      ...settings?.settings?.gitspace_config,
      scm: transformGitProvidersToPayload(formValues),
      ide: transformCodeEditorsToPayload(formValues, availableEditors)
    },
    infra_provider: transformCloudRegionsToPayload(formValues, settings)
  }
}
