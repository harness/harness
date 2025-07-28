import { useState, useMemo } from 'react'
import { useToaster } from '@harnessio/uicore'
import { useFindGitspaceSettings, useUpsertGitspaceSettings } from 'services/cde'
import { getIDETypeOptions } from 'cde-gitness/constants'
import { getErrorMessage } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import {
  createInitialValues,
  buildAdminSettingsPayload,
  type AdminSettingsFormValues
} from '../utils/adminSettingsUtils'

export const useAdminSettings = () => {
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const { accountInfo } = useAppContext()
  const [selectedTab, setSelectedTab] = useState('gitProviders')

  const {
    data: settings,
    loading: loadingSettings,
    error: errorSettings,
    refetch
  } = useFindGitspaceSettings({
    accountIdentifier: accountInfo?.identifier
  })

  const { mutate: upsertSettings, loading: loadingUpsert } = useUpsertGitspaceSettings({
    accountIdentifier: accountInfo?.identifier
  })

  const availableEditors = useMemo(() => getIDETypeOptions(getString), [getString])

  const tabs = useMemo(
    () => [
      { id: 'gitProviders', title: getString('cde.settings.gitProviders') },
      { id: 'codeEditors', title: getString('cde.settings.codeEditors') },
      { id: 'cloudRegions', title: getString('cde.settings.cloudRegionsAndMachineTypes') }
    ],
    [getString]
  )

  const initialValues = useMemo(() => createInitialValues(availableEditors), [availableEditors])

  const handleSave = async (values: AdminSettingsFormValues) => {
    try {
      const payload = buildAdminSettingsPayload(values, availableEditors, settings)
      await upsertSettings(payload)
      showSuccess(getString('cde.settings.saveSuccess'))
    } catch (err) {
      showError(getErrorMessage(err))
    }
  }

  const handleTabChange = (tabId: string) => {
    setSelectedTab(tabId)
  }

  return {
    settings,
    availableEditors,
    tabs,
    initialValues,
    selectedTab,
    loadingSettings,
    errorSettings,
    handleSave,
    handleTabChange,
    refetch,
    loadingUpsert
  }
}
