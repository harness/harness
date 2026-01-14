import React from 'react'
import { Card, Text, Layout, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { InfraProviderResource } from 'cde-gitness/utils/cloudRegionsUtils'
import InfraProviderCard from '../InfraProviderCard/InfraProviderCard'
import css from './InfraProviderPanel.module.scss'

interface InfraProviderPanelProps {
  infraProviderResources: InfraProviderResource[]
  selectedInfraProvider: string
  onInfraProviderClick: (providerKey: string) => void
}

const InfraProviderPanel: React.FC<InfraProviderPanelProps> = ({
  infraProviderResources,
  selectedInfraProvider,
  onInfraProviderClick
}) => {
  const { getString } = useStrings()

  return (
    <Card className={css.leftPanel}>
      <Layout.Vertical spacing={'small'}>
        <Text font={{ variation: FontVariation.H5 }} color={Color.GREY_800}>
          {getString('cde.settings.regions.availableCloudRegions')}
        </Text>
        <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500} className={css.panelSubtitle}>
          {getString('cde.settings.regions.availableCloudRegionsDesc')}
        </Text>

        <Container className={css.infraProviderList}>
          {infraProviderResources.map(provider => (
            <InfraProviderCard
              key={provider.key}
              provider={provider}
              isSelected={selectedInfraProvider === provider.key}
              onClick={onInfraProviderClick}
            />
          ))}
        </Container>
      </Layout.Vertical>
    </Card>
  )
}

export default InfraProviderPanel
