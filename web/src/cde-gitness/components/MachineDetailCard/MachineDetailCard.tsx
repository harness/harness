import React from 'react'
import { Container, Layout, Tag, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { regionType } from 'cde-gitness/constants'
import type { TypesCDEGateway } from 'services/cde'
import css from './MachineDetailCard.module.scss'

function MachineDetailCard({
  locationData,
  groupHealthData
}: {
  locationData: regionType
  groupHealthData?: TypesCDEGateway
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
            className={groupHealthData?.group_name ? css.rowContent : css.emptyRowContent}
            color={groupHealthData?.group_name ? Color.GREY_1000 : Color.GREY_500}>
            {groupHealthData?.group_name || 'N/A'}
          </Text>
        </Layout.Vertical>
        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayGroupHealth')}</Text>
          {!groupHealthData?.health && (
            <Text
              className={groupHealthData?.health ? css.rowContent : css.emptyRowContent}
              font={'small'}
              color={groupHealthData?.health ? Color.GREY_1000 : Color.GREY_500}>
              {!groupHealthData?.health && 'N/A'}
            </Text>
          )}
          {groupHealthData?.health === 'healthy' && <Tag intent="success">HEALTHY</Tag>}
          {groupHealthData?.health === 'unhealthy' && <Tag intent="danger">UNHEALTHY</Tag>}
        </Layout.Vertical>
        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.dnsManagedZone')}</Text>
          <Text className={css.rowContent} color={Color.GREY_1000}>
            {locationData?.dns}
          </Text>
        </Layout.Vertical>
        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayInstanceName')}</Text>
          <Text
            className={groupHealthData?.name ? css.rowContent : css.emptyRowContent}
            color={groupHealthData?.name ? Color.GREY_1000 : Color.GREY_500}>
            {groupHealthData?.name || 'N/A'}
          </Text>
        </Layout.Vertical>
        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.gatewayversionnumber')}</Text>
          <Text
            className={groupHealthData?.version ? css.rowContent : css.emptyRowContent}
            color={groupHealthData?.version ? Color.GREY_1000 : Color.GREY_500}>
            {groupHealthData?.version || 'N/A'}
          </Text>
        </Layout.Vertical>
        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.defaultSubnet')}</Text>
          <Text className={css.rowContent} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.default_subnet_ip_range}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.proxySubnet')}</Text>
          <Text className={css.rowContent} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.proxy_subnet_ip_range}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.configureInfra.domain')}</Text>
          <Text width={250} className={css.rowContent} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.domain}
          </Text>
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

export default MachineDetailCard
