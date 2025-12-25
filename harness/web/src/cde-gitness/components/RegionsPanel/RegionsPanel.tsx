import React, { useCallback } from 'react'
import { Text, Layout, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext } from 'formik'
import AWSIcon from 'cde-gitness/assests/aws.svg?url'
import type { InfraProviderResource, RegionData } from 'cde-gitness/utils/cloudRegionsUtils'
import { useStrings } from 'framework/strings'
import type { AdminSettingsFormValues } from 'cde-gitness/pages/AdminSettings/utils/adminSettingsUtils'
import GCPIcon from '../../../icons/google-cloud.svg?url'
import HarnessIcon from '../../../icons/Harness.svg?url'
import RegionAccordion from '../RegionAccordion/RegionAccordion'
import css from './RegionsPanel.module.scss'

interface RegionsPanelProps {
  selectedInfraProvider: string
  infraProviderResources: InfraProviderResource[]
  selectedProviderRegions: RegionData[]
  onRegionCheckboxChange: (infraProvider: string, region: string, checked: boolean) => void
  onMachineTypeChange: (infraProvider: string, region: string, machineTypeId: string, checked: boolean) => void
}

const RegionsPanel: React.FC<RegionsPanelProps> = ({
  selectedInfraProvider,
  infraProviderResources,
  selectedProviderRegions,
  onRegionCheckboxChange,
  onMachineTypeChange
}) => {
  const { values } = useFormikContext<AdminSettingsFormValues>()
  const { getString } = useStrings()

  const getProviderIcon = (providerKey: string) => {
    switch (providerKey) {
      case 'harness_gcp':
        return <img src={HarnessIcon} alt="Harness" height={36} />
      case 'hybrid_vm_gcp':
        return <img src={GCPIcon} alt="GCP" height={28} />
      case 'hybrid_vm_aws':
        return <img src={AWSIcon} alt="AWS" height={28} />
      default:
        return null
    }
  }

  const isRegionFullySelected = useCallback(
    (infraProvider: string, region: string): boolean => {
      const regionData = selectedProviderRegions.find(r => r.region === region)
      if (!regionData) return false

      return regionData.machine_types.every(
        machineType => values.cloudRegions?.[infraProvider]?.[region]?.[machineType.identifier] === true
      )
    },
    [selectedProviderRegions, values.cloudRegions]
  )

  const isRegionPartiallySelected = useCallback(
    (infraProvider: string, region: string): boolean => {
      const regionData = selectedProviderRegions.find(r => r.region === region)
      if (!regionData) return false

      const isFully = isRegionFullySelected(infraProvider, region)
      if (isFully) return false

      return regionData.machine_types.some(
        machineType => values.cloudRegions?.[infraProvider]?.[region]?.[machineType.identifier] === true
      )
    },
    [selectedProviderRegions, values.cloudRegions, isRegionFullySelected]
  )

  if (!selectedInfraProvider) {
    return null
  }

  return (
    <Container className={css.rightPanel}>
      <Layout.Vertical spacing={'small'}>
        <Layout.Horizontal spacing={'small'} className={css.headerWithIcon}>
          {getProviderIcon(selectedInfraProvider)}
          <Text font={{ variation: FontVariation.H5 }} color={Color.GREY_800}>
            {infraProviderResources.find(p => p.key === selectedInfraProvider)?.name}
          </Text>
        </Layout.Horizontal>
        <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500} className={css.panelSubtitle}>
          {getString('cde.settings.regions.selectRegionsDesc')}
        </Text>

        <Container className={css.regionsContainer}>
          {selectedProviderRegions.map(regionData => {
            const isFullySelected = isRegionFullySelected(selectedInfraProvider, regionData.region)
            const isPartiallySelected = isRegionPartiallySelected(selectedInfraProvider, regionData.region)
            const machineCount = regionData.machine_types.length
            const selectedCount = regionData.machine_types.filter(
              mt => values.cloudRegions?.[selectedInfraProvider]?.[regionData.region]?.[mt.identifier] === true
            ).length

            return (
              <RegionAccordion
                key={regionData.region}
                regionData={regionData}
                infraProvider={selectedInfraProvider}
                isFullySelected={isFullySelected}
                isPartiallySelected={isPartiallySelected}
                selectedCount={selectedCount}
                machineCount={machineCount}
                onRegionCheckboxChange={onRegionCheckboxChange}
                onMachineTypeChange={onMachineTypeChange}
              />
            )
          })}
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

export default RegionsPanel
