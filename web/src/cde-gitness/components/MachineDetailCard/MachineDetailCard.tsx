import React from 'react'
import { Icon } from '@harnessio/icons'
import { Container, Layout, Tag, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { regionType } from 'cde-gitness/constants'
import type { TypesCDEGateway } from 'services/cde'
import css from './MachineDetailCard.module.scss'

function MachineDetailCard({
  locationData,
  groupHealthData,
  loading
}: {
  locationData: regionType
  groupHealthData?: TypesCDEGateway
  loading?: boolean
}) {
  const { getString } = useStrings()
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
            tooltip={groupHealthData?.group_name}
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
          {groupHealthData?.health === 'healthy' && <Tag intent="success">HEALTHY</Tag>}
          {groupHealthData?.health === 'unhealthy' && <Tag intent="danger">UNHEALTHY</Tag>}
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayInstanceName')}</Text>
          <Text
            width={'90%'}
            icon={loading ? 'loading' : undefined}
            tooltip={groupHealthData?.name}
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
          {groupHealthData?.envoy_health === 'healthy' && <Tag intent="success">HEALTHY</Tag>}
          {groupHealthData?.envoy_health === 'unhealthy' && <Tag intent="danger">UNHEALTHY</Tag>}
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayversionnumber')}</Text>
          <Text
            icon={loading ? 'loading' : undefined}
            className={groupHealthData?.version ? css.rowContent : css.emptyRowContent}
            color={groupHealthData?.version ? Color.GREY_1000 : Color.GREY_500}>
            {loading ? '' : groupHealthData?.version || 'N/A'}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.defaultSubnet')}</Text>
          <Text className={css.rowContent} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.default_subnet_ip_range || 'N/A'}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.proxySubnet')}</Text>
          <Text className={css.rowContent} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.proxy_subnet_ip_range || 'N/A'}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.configureInfra.domain')}</Text>
          <Text width={250} className={css.rowContent} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.domain || 'N/A'}
          </Text>
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

export default MachineDetailCard
