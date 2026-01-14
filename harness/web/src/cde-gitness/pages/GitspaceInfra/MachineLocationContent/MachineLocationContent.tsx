import React, { useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  HarnessDocTooltip,
  Layout,
  Table,
  Tag,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { cloneDeep } from 'lodash-es'
import { Color, FontVariation } from '@harnessio/design-system'
import { HYBRID_VM_AWS, HYBRID_VM_GCP, type regionType } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage } from 'utils/Utils'
import MachineModal from 'cde-gitness/components/MachineModal/MachineModal'
import { useAppContext } from 'AppContext'
import {
  TypesInfraProviderResource,
  useDeleteInfraProviderResource,
  useListGateways,
  TypesInfraProviderConfig
} from 'services/cde'
import MachineDetailCard from 'cde-gitness/components/MachineDetailCard/MachineDetailCard'
import NoDataState from 'cde-gitness/components/NoDataState'
import AwsMachineModal from 'cde-gitness/components/MachineModal/AwsMachineModal'
import css from './MachineLocationContent.module.scss'

interface MachineLocationContentProps {
  locationData: regionType
  isConnected?: boolean
  machineData?: TypesInfraProviderResource[]
  infraprovider_identifier: string
  setRegionData: (val: Unknown) => void
  regionData: regionType[]
  provider: string
  infraDetails?: TypesInfraProviderConfig
  refetch: () => void
}

function MachineLocationContent({
  locationData,
  isConnected,
  machineData = [],
  infraprovider_identifier,
  setRegionData,
  regionData,
  provider,
  infraDetails,
  refetch
}: MachineLocationContentProps) {
  const { getString } = useStrings()
  const confirmDelete = useConfirmAct()
  const { accountInfo } = useAppContext()
  const [isOpen, setIsOpen] = useState(false)
  const { showError, showSuccess } = useToaster()
  const bpTableProps = { bordered: false, condensed: true, striped: true }

  const { data: gatewayResponse, loading: gatewayAPILoading } = useListGateways({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier,
    queryParams: { is_latest: 'true' } as any
  })

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
            refetch?.()
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
      <Layout.Vertical spacing={'xsmall'}>
        <Text color={Color.GREY_1000}>{persistent_disk_size} GB</Text>
        <Text color={Color.GREY_500}>{persistent_disk_type}</Text>
      </Layout.Vertical>
    )
  }

  function CustomDiskColumn(row: Unknown) {
    const { boot_disk_type, boot_disk_size } = row?.row?.original?.metadata
    return (
      <Layout.Vertical spacing={'xsmall'}>
        <Text color={Color.GREY_1000}>{boot_disk_size} GB</Text>
        <Text color={Color.GREY_500}>{boot_disk_type}</Text>
      </Layout.Vertical>
    )
  }

  function CustomImageColumn(row: Unknown) {
    const { vm_image_name, image_name, os = 'linux', arch = 'amd64' } = row?.row?.original?.metadata
    const displayValue = vm_image_name || image_name || ''

    const displayParts = displayValue.split('/')
    const firstLineDisplay = displayParts.length > 0 ? displayParts[displayParts.length - 1] : displayValue

    return (
      <Layout.Vertical spacing={'xsmall'}>
        <Text color={Color.GREY_1000} tooltip={displayValue} className={css.truncateText}>
          {firstLineDisplay}
        </Text>
        <Text color={Color.GREY_500}>{`${os}/${arch}`}</Text>
      </Layout.Vertical>
    )
  }

  function CustomMachineColumn(row: Unknown) {
    const { name } = row?.row?.original
    const { resource_name } = row?.row?.original?.metadata
    return <Text color={Color.GREY_1000}>{resource_name || name}</Text>
  }

  function CustomMachineTypeColumn(row: Unknown) {
    const { machine_type, cpu, memory } = row?.row?.original?.metadata
    return (
      <Layout.Vertical spacing={'xsmall'}>
        <Text color={Color.GREY_1000}>
          {' '}
          {cpu ? `${cpu} CPU cores` : ''}
          {memory ? `, ${memory} GB Memory` : ''}
        </Text>
        <Text color={Color.GREY_500}>{machine_type}</Text>
      </Layout.Vertical>
    )
  }

  const createMachineColumns = (providerType: string): any[] => {
    const isAws = providerType === HYBRID_VM_AWS
    const machineNameKey = isAws ? 'cde.Aws.instanceName' : 'cde.gitspaceInfraHome.machineName'
    const machineTypeKey = isAws ? 'cde.Aws.instanceType' : 'cde.gitspaceInfraHome.machineType'
    const zoneKey = isAws ? 'cde.Aws.availabilityZone' : 'cde.gitspaceInfraHome.zone'
    const imageKey = isAws ? 'cde.Aws.machineAmiId' : 'cde.gitspaceInfraHome.machineImageName'

    return [
      {
        Header: (
          <Layout.Horizontal>
            <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
              {getString(machineNameKey)}
            </Text>
            <HarnessDocTooltip tooltipId="InfraProviderResourceMachine" useStandAlone={true} />
          </Layout.Horizontal>
        ),
        Cell: CustomMachineColumn,
        accessor: 'name',
        width: '15%'
      },
      {
        Header: (
          <Layout.Horizontal>
            <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
              {getString(imageKey)}
            </Text>
            <HarnessDocTooltip tooltipId="InfraProviderResourceImage" useStandAlone={true} />
          </Layout.Horizontal>
        ),
        accessor: 'image',
        Cell: CustomImageColumn,
        width: '18%'
      },
      {
        Header: (
          <Layout.Horizontal>
            <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
              {getString(zoneKey)}
            </Text>
            <HarnessDocTooltip tooltipId="InfraProviderResourceZone" useStandAlone={true} />
          </Layout.Horizontal>
        ),
        accessor: 'metadata.zone',
        width: '13%'
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
        width: '16%'
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
        width: '16%'
      },
      {
        Header: (
          <Layout.Horizontal>
            <Text font={{ variation: FontVariation.TABLE_HEADERS }} className={css.headingText}>
              {getString(machineTypeKey)}
            </Text>
            <HarnessDocTooltip tooltipId="InfraProviderResourceMachine" useStandAlone={true} />
          </Layout.Horizontal>
        ),
        Cell: CustomMachineTypeColumn,
        accessor: 'type',
        width: '28%'
      },
      {
        Header: '',
        accessor: 'identifier',
        Cell: ActionCell,
        width: '3%'
      }
    ]
  }

  const gcpColumns = createMachineColumns(HYBRID_VM_GCP)
  const awsColumns = createMachineColumns(HYBRID_VM_AWS)

  const groupHealthData = gatewayResponse?.find(gateway => gateway.region === locationData.region_name)
  const MachineComponent = provider === HYBRID_VM_AWS ? AwsMachineModal : MachineModal

  return (
    <Container className={css.main}>
      <MachineComponent
        isOpen={isOpen}
        setIsOpen={setIsOpen}
        infraproviderIdentifier={infraprovider_identifier}
        regionIdentifier={locationData?.region_name}
        setRegionData={setRegionData}
        regionData={regionData}
        refetch={refetch}
      />
      <Layout.Vertical>
        <Container
          flex={{ justifyContent: 'flex-start', alignItems: 'center' }}
          margin={{ top: 'large', bottom: 'large' }}>
          <Text
            color={Color.BLACK}
            icon="globe-network"
            iconProps={{ size: 24, margin: { right: 'small' } }}
            font={{ size: 'medium' }}
            margin={{ right: 'medium' }}>
            {locationData.region_name}
          </Text>
          {gatewayAPILoading && <Icon name="loading" />}
          {!gatewayAPILoading && (
            <>
              {(!groupHealthData || (Array.isArray(groupHealthData) && groupHealthData.length === 0)) && (
                <Tag intent="danger">{getString('cde.gitspaceInfraHome.unhealthy')}</Tag>
              )}
              {groupHealthData?.overall_health === 'healthy' && (
                <Tag intent="success" className={css.filledTag}>
                  {getString('cde.gitspaceInfraHome.healthy')}
                </Tag>
              )}
              {groupHealthData?.overall_health === 'unhealthy' && (
                <Tag intent="danger" className={css.filledTag}>
                  {getString('cde.gitspaceInfraHome.unhealthy')}
                </Tag>
              )}
              {groupHealthData &&
                groupHealthData.overall_health &&
                !['healthy', 'unhealthy'].includes(groupHealthData.overall_health) && (
                  <Tag intent="warning" className={css.filledTag}>
                    {getString('cde.gitspaceInfraHome.unknown')}
                  </Tag>
                )}
            </>
          )}
        </Container>

        <MachineDetailCard
          loading={gatewayAPILoading}
          locationData={locationData}
          groupHealthData={groupHealthData}
          infraDetails={infraDetails}
        />

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
              {!isConnected ? (
                <Text className={css.addMachineNote}>{getString('cde.gitspaceInfraHome.addMachineNote')}</Text>
              ) : machineData?.length === 0 ? (
                <NoDataState type="machine" setIsOpen={setIsOpen} />
              ) : (
                <Table
                  columns={provider === HYBRID_VM_AWS ? awsColumns : gcpColumns}
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
