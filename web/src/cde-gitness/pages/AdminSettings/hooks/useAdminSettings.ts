import { useState, useMemo } from 'react'
import { useToaster } from '@harnessio/uicore'
import { useFindGitspaceSettings, useUpsertGitspaceSettings } from 'services/cde'
import { getErrorMessage } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import {
  createInitialValues,
  buildAdminSettingsPayload,
  type AdminSettingsFormValues,
  AdminSettingsTabs
} from '../utils/adminSettingsUtils'

export const useAdminSettings = () => {
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const { accountInfo } = useAppContext()
  const [selectedTab, setSelectedTab] = useState(AdminSettingsTabs.GIT_PROVIDERS)

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

  const tabs = useMemo(
    () => [
      { id: AdminSettingsTabs.GIT_PROVIDERS, title: getString('cde.settings.gitProviders') },
      { id: AdminSettingsTabs.CODE_EDITORS, title: getString('cde.settings.codeEditors') },
      { id: AdminSettingsTabs.CLOUD_REGIONS, title: getString('cde.settings.cloudRegionsAndMachineTypes') },
      { id: AdminSettingsTabs.GITSPACE_IMAGES, title: getString('cde.settings.gitspaceImages') }
    ],
    [getString]
  )

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const initialValues = useMemo(() => createInitialValues(settings, getString), [settings])

  const handleSave = async (values: AdminSettingsFormValues) => {
    try {
      const payload = buildAdminSettingsPayload(values, getString, settings)
      await upsertSettings(payload)
      showSuccess(getString('cde.settings.saveSuccess'))
      refetch()
    } catch (err) {
      showError(getErrorMessage(err))
    }
  }

  const handleTabChange = (tabId: AdminSettingsTabs) => {
    setSelectedTab(tabId)
  }

  return {
    settings,
    tabs,
    initialValues,
    selectedTab,
    loading: loadingSettings || loadingUpsert,
    errorSettings,
    handleSave,
    handleTabChange,
    refetch
  }
}
