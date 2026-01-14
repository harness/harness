import { useMemo } from 'react'
import type { UseStringsReturn } from 'framework/strings'
import type { TypesGitspaceSettingsResponse } from 'services/cde'
import type { EnumIDEType } from 'services/cde'
import type { IDEOption } from '../constants'

export const useFilteredIdeOptions = (
  ideOptions: IDEOption[],
  gitspaceSettings: TypesGitspaceSettingsResponse | null,
  getString: UseStringsReturn['getString']
): IDEOption[] =>
  useMemo(() => {
    if (!gitspaceSettings?.settings?.gitspace_config?.ide?.access_list) {
      return ideOptions
    }

    const { mode, list } = gitspaceSettings.settings.gitspace_config.ide.access_list

    if (mode === 'deny' && Array.isArray(list) && list.length > 0) {
      return ideOptions.filter(option => !list.includes(option.value as EnumIDEType))
    }

    return ideOptions
  }, [gitspaceSettings, ideOptions, getString])
