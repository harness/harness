import * as Yup from 'yup'
import type {
  TypesGitspaceSettingsData,
  TypesGitspaceSettingsResponse,
  TypesGitspaceRegionMachines,
  TypesInfraProviderSettings,
  TypesDevcontainerImage,
  EnumIDEType
} from 'services/cde'
import { scmOptionsCDE } from 'cde-gitness/pages/GitspaceCreate/CDECreateGitspace'
import { getIDETypeOptions, type IDEOption } from 'cde-gitness/constants'
import type { EnumInfraProviderType } from 'cde-gitness/services'
import type { StringsMap } from 'framework/strings/stringTypes'

export enum AdminSettingsTabs {
  GIT_PROVIDERS = 'gitProviders',
  CODE_EDITORS = 'codeEditors',
  CLOUD_REGIONS = 'cloudRegions',
  GITSPACE_IMAGES = 'gitspaceImages'
}

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
  gitspaceImages?: { default_image_added: boolean } & TypesDevcontainerImage
}
export const SSH_ACCESS_KEY = 'ssh_access'
/**
 * Creates initial form values for all admin settings tabs
 */
export const createInitialValues = (
  settings: TypesGitspaceSettingsResponse | null,
  getString: (key: keyof StringsMap) => string
): AdminSettingsFormValues => {
  const gitspace_config = settings?.settings?.gitspace_config || {}
  const devcontainer_image = settings?.settings?.gitspace_config?.devcontainer?.devcontainer_image || {}
  const availableEditors = getIDETypeOptions(getString)

  return {
    gitProviders: scmOptionsCDE.reduce((acc, provider) => {
      const isDenied = gitspace_config.scm?.access_list?.list?.includes(provider.value) ?? false
      acc[provider.value] = !isDenied
      return acc
    }, {} as { [key: string]: boolean }),
    codeEditors: availableEditors.reduce((acc, editor) => {
      const isDenied = gitspace_config.ide?.access_list?.list?.includes(editor.value) ?? false
      acc[editor.value] = !isDenied
      acc[SSH_ACCESS_KEY] = !(gitspace_config.ide?.disable_ssh === true)
      return acc
    }, {} as { [key: string]: boolean }),
    cloudRegions: {},
    gitspaceImages: {
      default_image_added: Boolean((devcontainer_image?.image_name?.trim().length ?? 0) > 0),
      image_connector_ref: devcontainer_image?.image_connector_ref,
      image_name: devcontainer_image?.image_name,
      access_list: devcontainer_image?.access_list
    }
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
  const deniedEditors: EnumIDEType[] = allEditors.filter(editor => !formValues.codeEditors[editor])
  return {
    access_list: {
      mode: 'deny' as const,
      list: deniedEditors
    },
    disable_ssh: formValues.codeEditors[SSH_ACCESS_KEY] === false
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
 * Transforms Gitspace Images form data to API payload format
 */
export const transformGitspaceImagesToPayload = (formValues: AdminSettingsFormValues) => {
  if (!formValues.gitspaceImages) {
    return {}
  }

  return {
    devcontainer_image: {
      access_list: {
        mode: 'allow' as const,
        list: formValues.gitspaceImages.access_list?.list || []
      },
      image_name: formValues.gitspaceImages.image_name,
      image_connector_ref: formValues.gitspaceImages.image_connector_ref
    }
  }
}

/**
 * Constructs the complete API payload for admin settings
 */
export const buildAdminSettingsPayload = (
  formValues: AdminSettingsFormValues,
  getString: (key: keyof StringsMap) => string,
  settings: TypesGitspaceSettingsResponse | null
): TypesGitspaceSettingsData => {
  return {
    ...settings?.settings,
    gitspace_config: {
      ...settings?.settings?.gitspace_config,
      scm: transformGitProvidersToPayload(formValues),
      ide: transformCodeEditorsToPayload(formValues, getIDETypeOptions(getString)),
      devcontainer: {
        ...settings?.settings?.gitspace_config?.devcontainer,
        ...transformGitspaceImagesToPayload(formValues)
      }
    },
    infra_provider: transformCloudRegionsToPayload(formValues, settings)
  }
}

const imagePattern = /^[^*]*\*?$/

export const getValidationSchema = (getString: (key: keyof StringsMap) => string) => {
  return Yup.object({
    gitspaceImages: Yup.object({
      access_list: Yup.object({
        list: Yup.array()
          .of(
            Yup.string()
              .required(getString('validation.imagePathIsRequired'))
              .matches(imagePattern, getString('validation.invalidImage'))
          )
          .notRequired()
      }).notRequired(),
      image_name: Yup.string().when('default_image_added', {
        is: true,
        then: (schema: Yup.StringSchema) =>
          schema.required(getString('validation.pathIsRequired')).trim().min(1, getString('validation.pathIsRequired')),
        otherwise: (schema: Yup.StringSchema) => schema.notRequired()
      })
    }).notRequired()
  })
}
