import React from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { TypesInfraProviderConfig } from 'services/cde'
import GCPIcon from '../../../icons/google-cloud.svg?url'
import css from './InfraDetailCard.module.scss'

function InfraDetailCard({
  infraDetails,
  regionCount
}: {
  infraDetails: TypesInfraProviderConfig
  regionCount: number
}) {
  const { getString } = useStrings()
  return (
    <Container className={css.infraDetailCard}>
      <Layout.Vertical spacing={'normal'}>
        <Layout.Horizontal spacing={'normal'}>
          <img src={GCPIcon} width={24} />
          <Text className={css.cardTitle}>{getString('cde.gitspaceInfraHome.infraDetails')}</Text>
        </Layout.Horizontal>
        <Layout.Horizontal className={css.cardContent} flex={{ justifyContent: 'space-between' }}>
          <Layout.Vertical>
            <Text className={css.rowHeader}>{getString('cde.configureInfra.name')}</Text>
            <Text className={css.rowContent}>{infraDetails?.metadata?.name}</Text>
          </Layout.Vertical>

          <Layout.Vertical>
            <Text className={css.rowHeader}>{getString('cde.configureInfra.domain')}</Text>
            <Text className={css.rowContent}>{infraDetails?.metadata?.domain}</Text>
          </Layout.Vertical>

          <Layout.Vertical>
            <Text className={css.rowHeader}>{getString('cde.configureInfra.machineType')}</Text>
            <Text className={css.rowContent}>{infraDetails?.metadata?.gateway?.machine_type}</Text>
          </Layout.Vertical>

          <Layout.Vertical>
            <Text className={css.rowHeader}>{getString('cde.configureInfra.numberOfInstance')}</Text>
            <Text className={css.rowContent}>{infraDetails?.metadata?.gateway?.instances}</Text>
          </Layout.Vertical>

          <Layout.Vertical>
            <Text className={css.rowHeader}>{getString('cde.configureInfra.numberOfLocations')}</Text>
            <Text className={css.rowContent}>{regionCount}</Text>
          </Layout.Vertical>
        </Layout.Horizontal>
      </Layout.Vertical>
    </Container>
  )
}

export default InfraDetailCard
