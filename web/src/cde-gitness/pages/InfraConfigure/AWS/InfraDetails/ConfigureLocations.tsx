import React, { useState } from 'react'
import { Container, HarnessDocTooltip, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'
import { Icon } from '@harnessio/icons'
import { cloneDeep } from 'lodash-es'
import { AwsRegionConfig, learnMoreRegion, type regionProp, ZoneConfig } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import RegionTable from 'cde-gitness/components/RegionTable/AwsRegionTable'
import NewRegionModal from './NewRegionModal'
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
}

const ConfigureLocations = ({ regionData, setRegionData }: LocationProps) => {
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
        {/* <Icon name="code-edit" size={24} onClick={() => openRegionModal(row?.row?.index)} /> */}
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
    /*
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
    */
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
            window.open(learnMoreRegion, '_blank')
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
      />
      <RegionTable columns={columns} addNewRegion={openRegionModal} regionData={regionData} />
    </Layout.Vertical>
  )
}

export default ConfigureLocations
