import React, { useState, useMemo } from 'react'
import { Icon } from '@harnessio/icons'
import { Container, Layout, Tag, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { regionType, AWSZoneConfig } from 'cde-gitness/constants'
import type { TypesCDEGateway, TypesInfraProviderConfig } from 'services/cde'
import { getTimeAgo } from 'cde-gitness/utils/time.utils'
import activityIcon from 'cde-gitness/assests/activity.svg?url'
import type { StringsMap } from '../../../framework/strings/stringTypes'
import { HYBRID_VM_AWS, HYBRID_VM_GCP } from '../../constants'
import css from './MachineDetailCard.module.scss'

interface DetailFieldProps {
  labelKey: keyof StringsMap
  value?: string | number | null
  provider?: string
  visibleFor?: string[]
  isLoading?: boolean
  width?: number
  className?: string
}

const DetailField: React.FC<DetailFieldProps> = ({
  labelKey,
  value,
  provider,
  visibleFor,
  isLoading = false,
  width,
  className
}) => {
  const { getString } = useStrings()

  if (visibleFor && provider && !visibleFor.includes(provider)) {
    return null
  }

  return (
    <Layout.Vertical spacing={'small'}>
      <Text className={css.rowHeader}>{getString(labelKey)}</Text>
      <Text
        width={width}
        icon={isLoading ? 'loading' : undefined}
        className={className || css.rowContent}
        iconProps={{ size: 14 }}
        color={Color.GREY_1000}>
        {isLoading ? '' : value || 'N/A'}
      </Text>
    </Layout.Vertical>
  )
}

interface ZoneTableHeaderProps {
  className?: string
}

const ZoneTableHeader: React.FC<ZoneTableHeaderProps> = ({ className }) => {
  const { getString } = useStrings()

  return (
    <div className={className || css.zoneDetailsTableHeader}>
      <div className={css.zoneDetailsTableCell}>
        <Text className={css.tableHeaderText}>{getString('cde.Aws.availabilityZone').toUpperCase()}</Text>
      </div>
      <div className={css.zoneDetailsTableCell}>
        <Text className={css.tableHeaderText}>{getString('cde.Aws.privateSubnetCidr').toUpperCase()}</Text>
      </div>
      <div className={css.zoneDetailsTableCell}>
        <Text className={css.tableHeaderText}>{getString('cde.Aws.publicSubnetCidr').toUpperCase()}</Text>
      </div>
    </div>
  )
}

interface EmptyZonesMessageProps {
  className?: string
}

const EmptyZonesMessage: React.FC<EmptyZonesMessageProps> = ({ className }) => {
  const { getString } = useStrings()

  return (
    <div className={className || css.zoneDetailsTableRow}>
      <div className={css.zoneDetailsTableCell} style={{ gridColumn: '1 / span 3' }}>
        <Text className={css.emptyRowContent} color={Color.GREY_500}>
          {getString('cde.gitspaceInfraHome.noZonesAvailable', { defaultValue: 'No zones available' })}
        </Text>
      </div>
    </div>
  )
}

interface ZoneTableRowProps {
  zoneConfig: AWSZoneConfig
  className?: string
}

const ZoneTableRow: React.FC<ZoneTableRowProps> = ({ zoneConfig, className }) => {
  return (
    <div className={className || css.zoneDetailsTableRow}>
      <div className={css.zoneDetailsTableCell}>
        <Text className={css.rowContent}>{zoneConfig.zone || 'N/A'}</Text>
      </div>
      <div className={css.zoneDetailsTableCell}>
        <Text className={css.rowContent}>{zoneConfig.private_cidr_block || 'N/A'}</Text>
      </div>
      <div className={css.zoneDetailsTableCell}>
        <Text className={css.rowContent}>{zoneConfig.public_cidr_block || 'N/A'}</Text>
      </div>
    </div>
  )
}

interface AwsZoneDetailsProps {
  zones: AWSZoneConfig[]
  isOpen: boolean
  onToggle: () => void
  className?: string
}

const AwsZoneDetails: React.FC<AwsZoneDetailsProps> = ({ zones, isOpen, onToggle, className }) => {
  const { getString } = useStrings()

  return (
    <Container className={className || css.zoneDetailsContainer}>
      <div className={css.collapsibleSection}>
        <div className={css.collapsibleHeader} onClick={onToggle}>
          <Layout.Horizontal spacing="small" flex>
            <Text
              className={css.sectionTitle}
              color={Color.BLACK}
              icon={isOpen ? 'chevron-down' : 'chevron-right'}
              iconProps={{ size: 16, margin: { right: 'small' } }}>
              {getString('cde.gitspaceInfraHome.zoneDetails')}
            </Text>
          </Layout.Horizontal>
        </div>

        {isOpen && (
          <div className={css.zoneDetailsContent}>
            <div className={css.zoneDetailsTable}>
              <ZoneTableHeader />

              {zones.length === 0 ? (
                <EmptyZonesMessage />
              ) : (
                zones.map((zoneConfig, index) => <ZoneTableRow key={index} zoneConfig={zoneConfig} />)
              )}
            </div>
          </div>
        )}
      </div>
    </Container>
  )
}

function MachineDetailCard({
  locationData,
  groupHealthData,
  loading,
  infraDetails
}: {
  locationData: regionType
  groupHealthData?: TypesCDEGateway
  loading?: boolean
  infraDetails?: TypesInfraProviderConfig
}) {
  const { getString } = useStrings()
  const [zoneDetailsOpen, setZoneDetailsOpen] = useState(true)

  const awsZones = useMemo(() => {
    if (infraDetails?.type !== HYBRID_VM_AWS || !infraDetails || !locationData?.region_name) {
      return []
    }

    const regionConfig = infraDetails?.metadata?.region_configs?.[locationData.region_name]
    return regionConfig?.availability_zones || []
  }, [infraDetails, locationData?.region_name])

  const gatewayAMIImage = infraDetails?.metadata?.region_configs?.[locationData?.region_name || '']?.gateway_ami_id
  const provider = infraDetails?.type || ''
  return (
    <Container className={css.locationDetail}>
      <Container className={css.machineHeader}>
        <Text className={css.cardTitle}>{getString('cde.gitspaceInfraHome.locationDetails')}</Text>
      </Container>

      <Container className={css.cardContent}>
        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayGroupName')}</Text>
          <Text
            width={'90%'}
            icon={loading ? 'loading' : undefined}
            className={groupHealthData?.group_name ? css.rowContent : css.emptyRowContent}
            color={groupHealthData?.group_name ? Color.GREY_1000 : Color.GREY_500}>
            {loading ? '' : groupHealthData?.group_name || 'N/A'}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayGroupHealth')}</Text>
          {!groupHealthData?.health && !loading && (
            <Text className={css.emptyRowContent} font={'small'} icon={undefined} color={Color.GREY_500}>
              N/A
            </Text>
          )}
          {loading && <Icon name="loading" />}
          {groupHealthData?.health === 'healthy' && (
            <Tag intent="success" className={css.filledTag}>
              {getString('cde.gitspaceInfraHome.healthy')}
            </Tag>
          )}
          {groupHealthData?.health === 'unhealthy' && (
            <Tag intent="danger" className={css.filledTag}>
              {getString('cde.gitspaceInfraHome.unhealthy')}
            </Tag>
          )}
          {groupHealthData && groupHealthData.health && !['healthy', 'unhealthy'].includes(groupHealthData.health) && (
            <Tag intent="warning" className={css.filledTag}>
              {getString('cde.gitspaceInfraHome.unknown')}
            </Tag>
          )}
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayInstanceName')}</Text>
          <Text
            width={'90%'}
            icon={loading ? 'loading' : undefined}
            className={groupHealthData?.name ? css.rowContent : css.emptyRowContent}
            color={groupHealthData?.name ? Color.GREY_1000 : Color.GREY_500}>
            {loading ? '' : groupHealthData?.name || 'N/A'}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.envoyHealth')}</Text>
          {!groupHealthData?.envoy_health && !loading && (
            <Text className={css.emptyRowContent} font={'small'} icon={undefined} color={Color.GREY_500}>
              N/A
            </Text>
          )}
          {loading && <Icon name="loading" />}
          {groupHealthData?.envoy_health === 'healthy' && (
            <Tag intent="success" className={css.filledTag}>
              {getString('cde.gitspaceInfraHome.healthy')}
            </Tag>
          )}
          {groupHealthData?.envoy_health === 'unhealthy' && (
            <Tag intent="danger" className={css.filledTag}>
              {getString('cde.gitspaceInfraHome.unhealthy')}
            </Tag>
          )}
          {groupHealthData &&
            groupHealthData.envoy_health &&
            !['healthy', 'unhealthy'].includes(groupHealthData.envoy_health) && (
              <Tag intent="warning" className={css.filledTag}>
                {getString('cde.gitspaceInfraHome.unknown')}
              </Tag>
            )}
        </Layout.Vertical>

        <DetailField
          labelKey="cde.gitspaceInfraHome.defaultSubnet"
          value={locationData?.default_subnet_ip_range}
          visibleFor={[HYBRID_VM_GCP]}
          isLoading={loading}
          provider={provider}
        />

        <DetailField
          labelKey="cde.gitspaceInfraHome.proxySubnet"
          value={locationData?.proxy_subnet_ip_range}
          visibleFor={[HYBRID_VM_GCP]}
          provider={provider}
          isLoading={loading}
        />

        <DetailField
          labelKey="cde.configureInfra.domain"
          value={locationData?.domain}
          provider={provider}
          isLoading={loading}
          className={css.domainOverFlow}
        />

        <DetailField
          labelKey="cde.Aws.gatewayAmiImage"
          value={gatewayAMIImage}
          visibleFor={[HYBRID_VM_AWS]}
          provider={provider}
          isLoading={loading}
          width={250}
        />

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.lastHeartbeat')}</Text>
          <Layout.Horizontal spacing={'xsmall'} className={css.rowContent}>
            <img src={activityIcon} alt="Activity" className={css.activityIcon} />
            <Text color={Color.GREY_1000}>
              {loading ? '' : groupHealthData?.updated ? getTimeAgo(groupHealthData.updated) : 'N/A'}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
      </Container>

      {infraDetails?.type === HYBRID_VM_AWS && (
        <AwsZoneDetails
          zones={awsZones}
          isOpen={zoneDetailsOpen}
          onToggle={() => setZoneDetailsOpen(!zoneDetailsOpen)}
        />
      )}
    </Container>
  )
}

export default MachineDetailCard
