import React, { useEffect, useState } from 'react'
import { Container, HarnessDocTooltip, Label, Layout, Select, Text, FormInput } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'
import { Icon } from '@harnessio/icons'
import { cloneDeep } from 'lodash-es'
import { learnMoreRegion, type regionProp, learMoreVMRunner } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import RegionTable from 'cde-gitness/components/RegionTable/RegionTable'
import NewRegionModal from './NewRegionModal'
import { InfraDetails } from './InfraDetails.constants'
import css from './InfraDetails.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<any>>

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
  regionData: regionProp[]
  setRegionData: (result: regionProp[]) => void
  initialData: regionProp
  runner: { region: string; zone: string; vm_image_name: string }
  setRunner: (result: { region: string; zone: string; vm_image_name: string }) => void
}

const ConfigureLocations = ({ regionData, setRegionData, runner, setRunner }: LocationProps) => {
  const { getString } = useStrings()
  const [isOpen, setIsOpen] = useState(false)
  const [editingRegion, setEditingRegion] = useState<regionProp | null>(null)
  const [isEditMode, setIsEditMode] = useState(false)

  const deleteRegion = (indx: number) => {
    const clonedData = cloneDeep(regionData)
    const result: regionProp[] = []
    clonedData.forEach((region: regionProp, index: number) => {
      if (index !== indx) {
        result.push(region)
      }
    })
    setRegionData(result)
  }

  const ActionCell: CellType = (row: any) => {
    return (
      <Container className={css.deleteContainer}>
        {/*<Icon name="code-edit" size={24} onClick={() => openRegionModal(row?.row?.index)} />*/}
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
  const columns = [
    {
      Header: (
        <Layout.Horizontal>
          <Text className={css.headingText}>{getString('cde.gitspaceInfraHome.region')}</Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionLocation" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: DisplayCell,
      accessor: 'location',
      placeholder: 'e.g us-west1',
      width: '20%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text className={css.headingText}>{getString('cde.gitspaceInfraHome.defaultSubnet')}</Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionDefaultSubnet" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: DisplayCell,
      accessor: 'defaultSubnet',
      placeholder: 'e.g 10.6.0.0/16',
      width: '15%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text className={css.headingText}>{getString('cde.gitspaceInfraHome.proxySubnet')}</Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionProxySubnet" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: DisplayCell,
      accessor: 'proxySubnet',
      placeholder: 'e.g 10.3.0.0/16',
      width: '15%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text className={css.headingText}>{getString('cde.configureInfra.domain')}</Text>
          <HarnessDocTooltip tooltipId="InfraProviderRegionDomain" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      Cell: DisplayCell,
      accessor: 'domain',
      placeholder: 'e.g us-west-ga.io',
      width: '25%'
    },
    {
      Header: '',
      accessor: 'identifier',
      Cell: ActionCell,
      width: '8%'
    }
  ]

  const addNewRegion = (data: regionProp) => {
    const clonedData: regionProp[] = cloneDeep(regionData)
    const payload: regionProp = {
      ...data,
      identifier: editingRegion?.identifier ?? clonedData.length + 1
    }

    const regionIndex =
      isEditMode && editingRegion ? clonedData.findIndex(r => r.identifier === editingRegion.identifier) : -1

    if (regionIndex !== -1) {
      clonedData[regionIndex] = payload
    } else {
      clonedData.push(payload)
    }

    setRegionData(clonedData)
    setIsOpen(false)
    setEditingRegion(null)
    setIsEditMode(false)
  }

  const openRegionModal = (regionIndex?: number) => {
    if (regionIndex !== undefined) {
      setEditingRegion(regionData[regionIndex])
      setIsEditMode(true)
    } else {
      setEditingRegion(null)
      setIsEditMode(false)
    }
    setIsOpen(true)
  }

  const runnerVMRegionOptions = regionData.map(item => ({ label: item.location, value: item.location }))

  // Get zones for the selected region
  const selectedRegion = runner?.region || ''
  const availableZones = selectedRegion
    ? InfraDetails.regions[selectedRegion as keyof typeof InfraDetails.regions] || []
    : []
  const runnerVMZoneOptions = availableZones.map(zone => ({ label: zone, value: zone }))

  // Reset zone selection if region changes or if the current zone doesn't belong to the selected region
  useEffect(() => {
    if (setRunner && runner?.zone) {
      if (!availableZones.includes(runner?.zone)) {
        setRunner({ ...runner, zone: '' })
      }
    }
  }, [selectedRegion, availableZones, runner?.zone, setRunner])
  return (
    <Layout.Vertical spacing="none" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.configureInfra.configureLocations')}</Text>
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
        onSubmit={addNewRegion}
        initialValues={editingRegion}
        isEditMode={isEditMode}
        existingRegions={regionData.map(region => region.location)}
      />
      <RegionTable columns={columns} addNewRegion={() => openRegionModal()} regionData={regionData} />
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
              runner?.zone && availableZones.includes(runner?.zone)
                ? runnerVMZoneOptions.find(zone => zone.value === runner?.zone) || null
                : null
            }
            onChange={value => {
              setRunner({ ...runner, zone: value?.value as string })
            }}
          />
        </Container>
        <Container>
          <FormInput.Text
            name="runner.vm_image_name"
            label={getString('cde.gitspaceInfraHome.machineImageName')}
            placeholder={getString('cde.gitspaceInfraHome.machineImageNamePlaceholder')}
            className={css.inputWithNote}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              // Also update local state for immediate UI updates
              setRunner({ ...runner, vm_image_name: e.target.value })
            }}
          />
          <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
            {getString('cde.configureInfra.defaultImageNoteText')}
          </Text>
        </Container>
      </Layout.Vertical>
    </Layout.Vertical>
  )
}

export default ConfigureLocations
