import React, { useState, useMemo } from 'react'
import { Layout, Container } from '@harnessio/uicore'
import { useFormikContext } from 'formik'
import type { TypesGitspaceSettingsResponse } from 'services/cde'
import InfraProviderPanel from 'cde-gitness/components/InfraProviderPanel/InfraProviderPanel'
import RegionsPanel from 'cde-gitness/components/RegionsPanel/RegionsPanel'
import { useInfraProviderResources } from 'cde-gitness/pages/AdminSettings/CloudRegions/hooks/useInfraProviderResources'
import {
  AdminSettingsFormValues,
  getCloudRegionFieldPath,
  processInfraProviderDenyList
} from 'cde-gitness/utils/cloudRegionsUtils'
import css from './CloudRegions.module.scss'

interface CloudRegionsProps {
  settings: TypesGitspaceSettingsResponse | null
}

const CloudRegions: React.FC<CloudRegionsProps> = ({ settings }) => {
  const { setFieldValue } = useFormikContext<AdminSettingsFormValues>()

  const infraProviderResources = useInfraProviderResources(settings)
  const [selectedInfraProvider, setSelectedInfraProvider] = useState<string>(infraProviderResources[0].key || '')

  React.useEffect(() => {
    if (settings && infraProviderResources.length > 0) {
      const newCloudRegionsValues = processInfraProviderDenyList(settings, infraProviderResources)
      setFieldValue('cloudRegions', newCloudRegionsValues, false)
    }
  }, [settings, infraProviderResources, setFieldValue])

  const selectedProviderRegions = useMemo(() => {
    if (!selectedInfraProvider) return []
    const provider = infraProviderResources.find(p => p.key === selectedInfraProvider)
    return provider?.regions || []
  }, [selectedInfraProvider, infraProviderResources])

  const handleInfraProviderClick = (providerKey: string) => {
    setSelectedInfraProvider(providerKey)
  }

  const handleMachineTypeChange = (infraProvider: string, region: string, machineTypeId: string, checked: boolean) => {
    const fieldPath = getCloudRegionFieldPath(infraProvider, region, machineTypeId)
    setFieldValue(fieldPath, checked)
  }

  const handleRegionCheckboxChange = (infraProvider: string, region: string, checked: boolean) => {
    const regionData = selectedProviderRegions.find(r => r.region === region)
    if (!regionData) return

    regionData.machine_types.forEach(machineType => {
      const fieldPath = getCloudRegionFieldPath(infraProvider, region, machineType.identifier)
      setFieldValue(fieldPath, checked)
    })
  }

  return (
    <Container className={css.container}>
      <Layout.Horizontal className={css.mainLayout}>
        <InfraProviderPanel
          infraProviderResources={infraProviderResources}
          selectedInfraProvider={selectedInfraProvider}
          onInfraProviderClick={handleInfraProviderClick}
        />
        <RegionsPanel
          selectedInfraProvider={selectedInfraProvider}
          infraProviderResources={infraProviderResources}
          selectedProviderRegions={selectedProviderRegions}
          onRegionCheckboxChange={handleRegionCheckboxChange}
          onMachineTypeChange={handleMachineTypeChange}
        />
      </Layout.Horizontal>
    </Container>
  )
}

export default CloudRegions
