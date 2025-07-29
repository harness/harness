import React, { useEffect, useState } from 'react'
import { Container, FormInput, HarnessDocTooltip, Label, Layout, Select, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'
import { Icon } from '@harnessio/icons'
import { cloneDeep } from 'lodash-es'
import {
  AwsRegionConfig,
  learnMoreRegionAws,
  type regionProp,
  ZoneConfig,
  learMoreVMRunner
} from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import RegionTable from 'cde-gitness/components/RegionTable/AwsRegionTable'
import NewRegionModal from './NewRegionModal'
import { InfraDetails } from './InfraDetails.constants'
import css from './InfraDetails.module.scss'

interface ExtendedAwsRegionConfig extends AwsRegionConfig {
  zones?: ZoneConfig[]
  identifier: number
}

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<ExtendedAwsRegionConfig>>

interface customCellProps {
  column: {
    id: string
    placeholder: string
  }
  row: {
    index: number
  }
  value: string
}

interface LocationProps {
  regionData: ExtendedAwsRegionConfig[]
  setRegionData: (result: ExtendedAwsRegionConfig[]) => void
  initialData: ExtendedAwsRegionConfig
  runner: { region: string; availability_zones: string; ami_id: string }
  setRunner: (result: { region: string; availability_zones: string; ami_id: string }) => void
}

const ConfigureLocations = ({ regionData, setRegionData, runner, setRunner }: LocationProps) => {
  const { getString } = useStrings()
  const [isOpen, setIsOpen] = useState(false)
  const [editingRegion, setEditingRegion] = useState<ExtendedAwsRegionConfig | null>(null)
  const [isEditMode, setIsEditMode] = useState(false)

  const deleteRegion = (regionIndex: number) => {
    const clonedData = cloneDeep(regionData)
    const result: ExtendedAwsRegionConfig[] = []
    clonedData.forEach((region: ExtendedAwsRegionConfig, index: number) => {
      if (index !== regionIndex) {
        result.push(region)
      }
    })
    setRegionData(result)
  }

  const ActionCell: CellType = (row: any) => {
    return (
      <Container className={css.deleteContainer}>
        <Icon name="code-edit" size={24} onClick={() => openRegionModal(row?.row?.index)} />
        <Icon name="code-delete" size={24} onClick={() => deleteRegion(row?.row?.index)} />
      </Container>
    )
  }

  const DisplayCell: any = (row: customCellProps) => {
    return (
      <Container className={css.inputContainer}>
        <Text>{row?.value || '-'}</Text>
      </Container>
    )
  }

  const ZoneCountCell: any = ({ row }: { row: { original: regionProp } }) => {
    const region = row.original
    const zoneCount = region.zones?.length || 0

    return (
      <Container className={css.inputContainer}>
        <Text>{zoneCount}</Text>
      </Container>
    )
  }

  const columns = [
    {
      Header: '',
      Cell: <></>,
      accessor: 'toggle',
      width: '5%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} color={Color.GREY_700}>
            {getString('cde.gitspaceInfraHome.region')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionLocation" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: DisplayCell,
      accessor: 'region_name',
      width: '27%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} color={Color.GREY_700}>
            {getString('cde.Aws.gatewayAmi')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionAmi" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: DisplayCell,
      accessor: 'gateway_ami_id',
      width: '21%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} color={Color.GREY_700}>
            {getString('cde.Aws.availabilityZone') + 's'}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionZones" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: ZoneCountCell,
      accessor: 'zones',
      width: '26%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} color={Color.GREY_700}>
            {getString('cde.configureInfra.domain')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionDomain" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: DisplayCell,
      accessor: 'domain',
      width: '36%'
    },
    {
      Header: '',
      accessor: 'identifier',
      Cell: ActionCell,
      width: '7%'
    }
  ]

  const addNewRegion = (data: regionProp) => {
    const clonedData: ExtendedAwsRegionConfig[] = cloneDeep(regionData)

    const basePayload: ExtendedAwsRegionConfig = {
      region_name: data.location,
      gateway_ami_id: data.gatewayAmiId || '',
      domain: data.domain,
      private_cidr_block: data.zones?.[0]?.privateSubnet || '',
      public_cidr_block: data.zones?.[0]?.publicSubnet || '',
      zone: data.zones?.[0]?.zone || '',
      zones: data.zones,
      identifier: editingRegion?.identifier ?? clonedData.length + 1
    }

    const regionIndex =
      isEditMode && editingRegion ? clonedData.findIndex(r => r.identifier === editingRegion.identifier) : -1

    if (regionIndex !== -1) {
      clonedData[regionIndex] = basePayload
    } else {
      clonedData.push(basePayload)
    }

    setRegionData(clonedData)
    setIsOpen(false)
    setIsEditMode(false)
    setEditingRegion(null)
  }

  const openRegionModal = (index?: number) => {
    if (index !== undefined && regionData[index]) {
      const region = regionData[index]
      const regionPropValue: regionProp = {
        location: region.region_name,
        gatewayAmiId: region.gateway_ami_id,
        domain: region.domain,
        defaultSubnet: region.private_cidr_block || '',
        proxySubnet: region.public_cidr_block || '',
        identifier: region.identifier,
        zones: region.zones
      }
      setEditingRegion(regionPropValue as unknown as ExtendedAwsRegionConfig)
      setIsEditMode(true)
    } else {
      // Add mode
      setEditingRegion(null)
      setIsEditMode(false)
    }
    setIsOpen(true)
  }

  const runnerVMRegionOptions = regionData.map(item => ({ label: item.region_name, value: item.region_name }))

  // Get zones for the selected region
  const selectedRegion = runner?.region || ''
  const availableZones = selectedRegion
    ? InfraDetails.regions[selectedRegion as keyof typeof InfraDetails.regions] || []
    : []
  const runnerVMZoneOptions = availableZones.map(zone => ({ label: zone, value: zone }))

  // Reset zone selection if region changes or if the current zone doesn't belong to the selected region
  useEffect(() => {
    if (setRunner && runner?.availability_zones) {
      if (!availableZones.includes(runner?.availability_zones)) {
        setRunner({ ...runner, availability_zones: '' })
      }
    }
  }, [selectedRegion, availableZones, runner?.availability_zones, setRunner])
  return (
    <Layout.Vertical spacing="none" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.Aws.configureRegionsAndZones')}</Text>
      <Layout.Horizontal spacing="small" className={css.bottomSpacing}>
        <Text color={Color.GREY_400} className={css.headerLinkText}>
          {getString('cde.configureInfra.configureLocationNote')}
        </Text>
        <Text
          color={Color.PRIMARY_7}
          className={css.headerLinkText}
          onClick={() => {
            window.open(learnMoreRegionAws, '_blank')
          }}>
          {getString('cde.configureInfra.learnMore')}
        </Text>
      </Layout.Horizontal>

      <NewRegionModal
        isOpen={isOpen}
        setIsOpen={setIsOpen}
        onSubmit={formData => {
          addNewRegion(formData as regionProp)
        }}
        initialValues={editingRegion as regionProp | null}
        isEditMode={isEditMode}
        existingRegions={regionData.map(region => region.region_name)}
      />
      <RegionTable columns={columns} addNewRegion={openRegionModal} regionData={regionData} />
      <br />
      <Text className={css.basicDetailsHeading}>{getString('cde.gitspaceInfraHome.configureVMRunnerImage')}</Text>
      <Layout.Horizontal spacing="small">
        <Text color={Color.GREY_400} className={css.headerLinkText}>
          {getString('cde.gitspaceInfraHome.configureVMRunnerImageNote')}
        </Text>
        <Text
          color={Color.PRIMARY_7}
          className={css.headerLinkText}
          onClick={() => {
            window.open(learMoreVMRunner, '_blank')
          }}>
          {getString('cde.configureInfra.learnMore')}
        </Text>
      </Layout.Horizontal>
      <Layout.Vertical className={css.regionContainer} spacing="large">
        <Container>
          <Label className={css.runnerregion}>{getString('cde.gitspaceInfraHome.runnerVMRegion')}</Label>
          <Select
            addClearBtn
            name={getString('cde.gitspaceInfraHome.runnerVMRegion')}
            items={runnerVMRegionOptions}
            value={
              runner?.region ? runnerVMRegionOptions.find(region => region.value === runner?.region) || null : null
            }
            onChange={value => {
              setRunner({ ...runner, region: value?.value as string })
            }}
          />
        </Container>
        <Container>
          <Label className={css.runnerregion}>{getString('cde.gitspaceInfraHome.runnerVMZone')}</Label>
          <Select
            addClearBtn
            name={getString('cde.gitspaceInfraHome.runnerVMZone')}
            items={runnerVMZoneOptions}
            disabled={!selectedRegion}
            value={
              runner?.availability_zones && availableZones.includes(runner?.availability_zones)
                ? runnerVMZoneOptions.find(zone => zone.value === runner?.availability_zones) || null
                : null
            }
            onChange={value => {
              setRunner({ ...runner, availability_zones: value?.value as string })
            }}
          />
        </Container>
        <Container>
          <FormInput.Text
            name="runner.ami_id"
            label={getString('cde.Aws.runnerAmiId')}
            placeholder={getString('cde.Aws.runnerAmiIdPlaceholder')}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              // Also update local state for immediate UI updates
              setRunner({ ...runner, ami_id: e.target.value })
            }}
          />
        </Container>
      </Layout.Vertical>
    </Layout.Vertical>
  )
}

export default ConfigureLocations
