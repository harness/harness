import React from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { StringKeys } from 'framework/strings'
import type { TypesInfraProviderConfig } from 'services/cde'
import AWSIcon from 'cde-gitness/assests/aws.svg?url'
import { HYBRID_VM_GCP, HYBRID_VM_AWS } from 'cde-gitness/constants'
import GCPIcon from '../../../icons/google-cloud.svg?url'
import css from './InfraDetailCard.module.scss'

interface InfraDetailCardProps {
  infraDetails: TypesInfraProviderConfig
  regionCount: number
  provider: string
}

type CardField = {
  stringKey: string
  value: string | number | undefined
}

function InfraDetailCard({ infraDetails, regionCount, provider }: InfraDetailCardProps) {
  const { getString } = useStrings()

  const providerConfigs: Record<string, { icon: string; fields: CardField[] }> = {
    [HYBRID_VM_GCP]: {
      icon: GCPIcon,
      fields: [
        { stringKey: 'cde.configureInfra.name', value: infraDetails?.metadata?.name },
        { stringKey: 'cde.configureInfra.domain', value: infraDetails?.metadata?.domain },
        { stringKey: 'cde.configureInfra.machineType', value: infraDetails?.metadata?.gateway?.machine_type },
        { stringKey: 'cde.configureInfra.numberOfInstance', value: infraDetails?.metadata?.gateway?.instances },
        { stringKey: 'cde.configureInfra.numberOfLocations', value: regionCount }
      ]
    },
    [HYBRID_VM_AWS]: {
      icon: AWSIcon,
      fields: [
        { stringKey: 'cde.configureInfra.name', value: infraDetails?.metadata?.name },
        { stringKey: 'cde.Aws.VpcCidrBlock', value: infraDetails?.metadata?.vpc_cidr_block },
        { stringKey: 'cde.configureInfra.domain', value: infraDetails?.metadata?.domain },
        { stringKey: 'cde.Aws.instanceType', value: infraDetails?.metadata?.gateway?.instance_type },
        { stringKey: 'cde.Aws.numberOfRegions', value: regionCount }
      ]
    }
  }

  const currentConfig = providerConfigs[provider] || providerConfigs[HYBRID_VM_GCP]

  const renderCardFields = (fields: CardField[]) => {
    return fields.map((field, index) => (
      <Layout.Vertical key={index}>
        <Text className={css.rowHeader}>{getString(field.stringKey as StringKeys)}</Text>
        <Text className={css.rowContent}>{field.value}</Text>
      </Layout.Vertical>
    ))
  }

  return (
    <Container className={css.infraDetailCard}>
      <Layout.Vertical spacing={'normal'}>
        <Layout.Horizontal spacing={'normal'}>
          <img src={currentConfig.icon} width={24} />
          <Text className={css.cardTitle}>{getString('cde.gitspaceInfraHome.infraDetails')}</Text>
        </Layout.Horizontal>
        <Layout.Horizontal className={css.cardContent} flex={{ justifyContent: 'space-between' }}>
          {renderCardFields(currentConfig.fields)}
        </Layout.Horizontal>
      </Layout.Vertical>
    </Container>
  )
}

export default InfraDetailCard
