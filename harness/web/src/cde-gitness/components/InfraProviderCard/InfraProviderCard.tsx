import React from 'react'
import { Card, Text, Layout, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import AWSIcon from 'cde-gitness/assests/aws.svg?url'
import type { InfraProviderResource } from 'cde-gitness/utils/cloudRegionsUtils'
import GCPIcon from '../../../icons/google-cloud.svg?url'
import HarnessIcon from '../../../icons/Harness.svg?url'
import css from './InfraProviderCard.module.scss'

interface InfraProviderCardProps {
  provider: InfraProviderResource
  isSelected: boolean
  onClick: (providerKey: string) => void
}

const InfraProviderCard: React.FC<InfraProviderCardProps> = ({ provider, isSelected, onClick }) => {
  const getProviderIcon = (providerKey: string) => {
    switch (providerKey) {
      case 'harness_gcp':
        return <img src={HarnessIcon} alt="Harness" height={36} width={32} />
      case 'hybrid_vm_gcp':
        return <img src={GCPIcon} alt="GCP" height={24} />
      case 'hybrid_vm_aws':
        return <img src={AWSIcon} alt="AWS" height={36} width={32} />
      default:
        return <div className={css.providerIcon}></div>
    }
  }

  return (
    <Card
      className={`${css.infraProviderCard} ${isSelected ? css.selected : ''}`}
      onClick={() => onClick(provider.key)}>
      <Layout.Horizontal className={css.infraProviderContent}>
        <Container className={css.infraProviderIcon}>{getProviderIcon(provider.key)}</Container>
        <Container className={css.infraProviderInfo}>
          <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.GREY_800}>
            {provider.name}
          </Text>
          <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
            {provider.regions.length} regions available
          </Text>
        </Container>
      </Layout.Horizontal>
    </Card>
  )
}

export default InfraProviderCard
