import React from 'react'
import { Card, Text, Layout, Checkbox, Container, Accordion } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext } from 'formik'
import FlagsIcon from 'cde-gitness/assests/Flags.svg?url'
import { useStrings } from 'framework/strings'
import type { RegionData, AdminSettingsFormValues } from 'cde-gitness/utils/cloudRegionsUtils'
import MachineTypeCard from '../MachineTypeCard/MachineTypeCard'
import css from './RegionAccordion.module.scss'

interface RegionAccordionProps {
  regionData: RegionData
  infraProvider: string
  isFullySelected: boolean
  isPartiallySelected: boolean
  selectedCount: number
  machineCount: number
  onRegionCheckboxChange: (infraProvider: string, region: string, checked: boolean) => void
  onMachineTypeChange: (infraProvider: string, region: string, machineTypeId: string, checked: boolean) => void
}

const RegionAccordion: React.FC<RegionAccordionProps> = ({
  regionData,
  infraProvider,
  isFullySelected,
  isPartiallySelected,
  selectedCount,
  machineCount,
  onRegionCheckboxChange,
  onMachineTypeChange
}) => {
  const { values } = useFormikContext<AdminSettingsFormValues>()
  const { getString } = useStrings()

  return (
    <Card className={css.infraProviderCard}>
      <Accordion summaryClassName={css.accordionSummary} detailsClassName={css.accordionDetails}>
        <Accordion.Panel
          className={css.accordionPanel}
          id={regionData.region}
          summary={
            <Layout.Horizontal className={css.regionHeader}>
              <Checkbox
                checked={isPartiallySelected || isFullySelected}
                className={css.regionCheckbox}
                disabled={true}
              />
              <img src={FlagsIcon} alt="Flags" height={40} />
              <Layout.Horizontal className={css.regionHeaderDetails}>
                <Layout.Vertical spacing={'xsmall'}>
                  <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
                    {getString('cde.settings.regions.region')}
                  </Text>
                  <Layout.Horizontal spacing={'small'}>
                    <Text font={{ variation: FontVariation.CARD_TITLE }} color={Color.GREY_700}>
                      {regionData.region}
                    </Text>
                    {regionData.region_display_name !== '' && (
                      <Text font={{ variation: FontVariation.CARD_TITLE }} color={Color.GREY_700}>
                        ({regionData.region_display_name})
                      </Text>
                    )}
                  </Layout.Horizontal>
                </Layout.Vertical>
                <div className={css.machinesCountTag}>
                  <Text font={{ variation: FontVariation.BODY2 }} color={Color.PRIMARY_7}>
                    {selectedCount}/{machineCount} machines selected
                  </Text>
                </div>
              </Layout.Horizontal>
            </Layout.Horizontal>
          }
          details={
            <Container className={css.machineTypesContainer}>
              <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500} className={css.machineTypesTitle}>
                {getString('cde.settings.regions.selectMachineTypes')}
              </Text>

              <Layout.Horizontal className={css.selectAllContainer}>
                <Checkbox
                  checked={isFullySelected}
                  indeterminate={!isFullySelected && isPartiallySelected}
                  onChange={e => onRegionCheckboxChange(infraProvider, regionData.region, e.currentTarget.checked)}
                  className={css.selectAllCheckbox}
                />
                <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.GREY_800}>
                  {getString('cde.settings.regions.selectAllMachineTypes')}
                </Text>
              </Layout.Horizontal>

              <Container className={css.machineTypesList}>
                {regionData.machine_types.map(machineType => (
                  <MachineTypeCard
                    key={machineType.identifier}
                    machineType={machineType}
                    checked={
                      values.cloudRegions?.[infraProvider]?.[regionData.region]?.[machineType.identifier] === true
                    }
                    onChange={checked =>
                      onMachineTypeChange(infraProvider, regionData.region, machineType.identifier, checked)
                    }
                  />
                ))}
              </Container>
            </Container>
          }
        />
      </Accordion>
    </Card>
  )
}

export default RegionAccordion
