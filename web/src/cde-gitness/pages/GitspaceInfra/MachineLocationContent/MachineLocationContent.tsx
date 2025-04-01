import React, { useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  HarnessDocTooltip,
  Layout,
  Table,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { cloneDeep } from 'lodash-es'
import { Color, FontVariation } from '@harnessio/design-system'
import type { regionType } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage } from 'utils/Utils'
import MachineModal from 'cde-gitness/components/MachineModal/MachineModal'
import { useAppContext } from 'AppContext'
import { TypesInfraProviderResource, useDeleteInfraProviderResource } from 'services/cde'
import NoMachineCard from 'cde-gitness/components/NoMachineCard/NoMachineCard'
import MachineDetailCard from 'cde-gitness/components/MachineDetailCard/MachineDetailCard'
import NoMachineIcon from '../../../../icons/NoMachine.svg?url'
import css from './MachineLocationContent.module.scss'

interface MachineLocationContentProps {
  locationData: regionType
  isConnected?: boolean
  machineData?: TypesInfraProviderResource[]
  infraprovider_identifier: string
  setRegionData: (val: Unknown) => void
  regionData: regionType[]
}

function MachineLocationContent({
  locationData,
  isConnected,
  machineData = [],
  infraprovider_identifier,
  setRegionData,
  regionData
}: MachineLocationContentProps) {
  const { getString } = useStrings()
  const confirmDelete = useConfirmAct()
  const { accountInfo } = useAppContext()
  const [isOpen, setIsOpen] = useState(false)
  const { showError, showSuccess } = useToaster()
  const bpTableProps = { bordered: false, condensed: true, striped: true }

  function ActionCell(row: Unknown) {
    const { mutate: deleteResource } = useDeleteInfraProviderResource({
      accountIdentifier: accountInfo?.identifier,
      infraprovider_resource_identifier: row?.row?.values?.identifier,
      infraprovider_identifier
    })

    const deleteMachine = (e: React.MouseEvent, rowData: Unknown) => {
      confirmDelete({
        intent: 'danger',
        title: `${getString('cde.gitspaceInfraHome.deleteMachineTitle', { name: rowData?.row?.values?.name })}`,
        message: getString('cde.gitspaceInfraHome.deleteInfraText'),
        confirmText: getString('delete'),
        action: async () => {
          try {
            e.preventDefault()
            e.stopPropagation()
            await deleteResource('')
            showSuccess(getString('cde.deleteMachineSuccess'))
            const cloneData = cloneDeep(regionData)
            const updatedData: regionType[] = []
            cloneData?.forEach((region: regionType) => {
              if (region?.region_name === locationData?.region_name) {
                region.machines = region?.machines?.filter((mac: any) => mac?.identifier !== rowData?.value)
              }
              updatedData.push(region)
            })
            setRegionData(updatedData)
          } catch (exception) {
            showError(getErrorMessage(exception))
          }
        }
      })
    }

    return (
      <Container className={css.deleteContainer}>
        <Icon name="code-delete" size={24} onClick={e => deleteMachine(e, row)} />
      </Container>
    )
  }

  function CustomPersistentDiskColumn(row: Unknown) {
    const { persistent_disk_size, persistent_disk_type } = row?.row?.original?.metadata
    return (
      <Text color={Color.GREY_1000}>
        {persistent_disk_size}GB ({persistent_disk_type})
      </Text>
    )
  }

  function CustomDiskColumn(row: Unknown) {
    const { boot_disk_type, boot_disk_size } = row?.row?.original?.metadata
    return (
      <Text color={Color.GREY_1000}>
        {boot_disk_size}GB ({boot_disk_type})
      </Text>
    )
  }

  const columns: any = [
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
            {getString('cde.gitspaceInfraHome.machine')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderResourceMachine" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      accessor: 'name',
      width: '16%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
            {getString('cde.gitspaceInfraHome.persistentDisk')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderResourceDiskType" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      accessor: 'disk',
      Cell: CustomPersistentDiskColumn,
      width: '17%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
            {getString('cde.gitspaceInfraHome.zone')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderResourceZone" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      accessor: 'metadata.zone',
      width: '12%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
            {getString('cde.gitspaceInfraHome.bootDisk')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderResourceBootSize" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      accessor: 'metadata.boot_disk_size',
      Cell: CustomDiskColumn,
      width: '20%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
            {getString('cde.gitspaceInfraHome.cpu')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderResourceCPU" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      accessor: 'cpu',
      width: '15%'
    },
    {
      Header: (
        <Layout.Horizontal>
          <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
            {getString('cde.gitspaceInfraHome.memoryInGb')}
          </Text>
          <HarnessDocTooltip tooltipId="InfraProviderResourceMemory" useStandAlone={true} />
        </Layout.Horizontal>
      ),
      accessor: 'memory',
      width: '15%'
    },
    {
      Header: '',
      accessor: 'identifier',
      Cell: ActionCell,
      width: '5%'
    }
  ]

  return (
    <Container className={css.main}>
      <MachineModal
        isOpen={isOpen}
        setIsOpen={setIsOpen}
        infraproviderIdentifier={infraprovider_identifier}
        regionIdentifier={locationData?.region_name}
        setRegionData={setRegionData}
        regionData={regionData}
      />
      <Layout.Vertical>
        <MachineDetailCard locationData={locationData} />

        <Container className={css.machineDetail}>
          <Layout.Horizontal spacing={'none'} className={css.machineHeader} flex={{ justifyContent: 'space-between' }}>
            <Layout.Horizontal spacing={'none'}>
              <Text className={css.cardTitle}>{getString('cde.gitspaceInfraHome.machines')}</Text>
              <Text className={css.countLabel}>{machineData?.length ?? 0}</Text>
            </Layout.Horizontal>
            {machineData?.length > 0 ? (
              <Button
                text={getString('cde.gitspaceInfraHome.newMachine')}
                icon="plus"
                variation={ButtonVariation.SECONDARY}
                onClick={() => setIsOpen(true)}
              />
            ) : (
              <></>
            )}
          </Layout.Horizontal>

          <Container className={css.emptyMachineCard}>
            <Layout.Horizontal className={css.messageContainer}>
              {machineData?.length === 0 ? <img className={css.noMachineIcon} src={NoMachineIcon} /> : <></>}
              {!isConnected ? (
                <Text className={css.addMachineNote}>{getString('cde.gitspaceInfraHome.addMachineNote')}</Text>
              ) : machineData?.length === 0 ? (
                <NoMachineCard setIsOpen={setIsOpen} />
              ) : (
                <Table
                  columns={columns}
                  bpTableProps={bpTableProps}
                  className={css.tableContainer}
                  data={machineData}
                />
              )}
            </Layout.Horizontal>
          </Container>
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

export default MachineLocationContent
