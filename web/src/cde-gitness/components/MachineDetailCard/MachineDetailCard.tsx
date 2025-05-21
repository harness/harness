import React from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { regionType } from 'cde-gitness/constants'
import css from './MachineDetailCard.module.scss'

function MachineDetailCard({ locationData }: { locationData: regionType }) {
  const { getString } = useStrings()
  return (
    <Container className={css.locationDetail}>
      <Container className={css.machineHeader}>
        <Text className={css.cardTitle}>{getString('cde.gitspaceInfraHome.locationDetails')}</Text>
      </Container>
      <Layout.Horizontal className={css.cardContent} flex={{ justifyContent: 'space-between' }}>
        <Layout.Vertical spacing={'small'}>
          <Text className={css.rowHeader}>{getString('cde.gitspaceInfraHome.locationName')}</Text>
          <Text className={css.rowContent} icon={'globe-network'} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.region_name}
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
          <Text className={css.rowContent} iconProps={{ size: 14 }} color={Color.GREY_1000}>
            {locationData?.domain}
          </Text>
        </Layout.Vertical>
      </Layout.Horizontal>
    </Container>
  )
}

export default MachineDetailCard
