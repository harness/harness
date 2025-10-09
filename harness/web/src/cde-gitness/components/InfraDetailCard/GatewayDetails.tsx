import React from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { HYBRID_VM_GCP } from 'cde-gitness/constants'
import type { TypesInfraProviderConfig } from 'services/cde'
import CopyToClipboard from '../../../components/CopyToClipBoard/CopyToClipBoard'
import { getTruncatedValue } from '../../utils/helper.utils'
import css from './InfraDetailCard.module.scss'

interface GatewayDetailsProps {
  infraDetails: TypesInfraProviderConfig
  provider: string
  initialOpen?: boolean
}

const GatewayDetails: React.FC<GatewayDetailsProps> = ({ infraDetails, provider, initialOpen = true }) => {
  const [isOpen, setIsOpen] = React.useState(initialOpen)
  const { getString } = useStrings()

  return (
    <Container className={css.collapsibleSection}>
      <Container className={css.collapsibleHeader} onClick={() => setIsOpen(!isOpen)}>
        <Layout.Horizontal spacing="small" flex>
          <Text
            className={css.sectionTitle}
            color={Color.BLACK}
            icon={isOpen ? 'chevron-down' : 'chevron-right'}
            iconProps={{ size: 16, margin: { right: 'small' } }}>
            {getString('cde.configureInfra.gatewayDetails')}
          </Text>
        </Layout.Horizontal>
      </Container>

      {isOpen && (
        <Container className={css.collapsibleContent}>
          <Container className={css.detailsGrid}>
            <Container className={css.detailsGridItem}>
              <Text className={css.rowHeader}>{getString('cde.configureInfra.machineType')}</Text>
              <Text className={css.rowContent}>
                {provider === HYBRID_VM_GCP
                  ? infraDetails?.metadata?.gateway?.machine_type || ''
                  : infraDetails?.metadata?.gateway?.instance_type || ''}
              </Text>
            </Container>
            {provider === HYBRID_VM_GCP && (
              <Container className={css.detailsGridItem}>
                <Text className={css.rowHeader}>{getString('cde.configureInfra.machineImageName')}</Text>
                <Layout.Horizontal className={css.imageNameContainer} spacing="small">
                  <Text
                    className={`${css.rowContent} ${css.truncateText}`}
                    tooltip={infraDetails?.metadata?.gateway?.vm_image_name || ''}>
                    {getTruncatedValue(infraDetails?.metadata?.gateway?.vm_image_name || '')}
                  </Text>
                  <Container className={css.copyButtonContainer}>
                    <CopyToClipboard
                      content={infraDetails?.metadata?.gateway?.vm_image_name || ''}
                      iconSize={16}
                      className={css.copyButton}
                    />
                  </Container>
                </Layout.Horizontal>
              </Container>
            )}
          </Container>
        </Container>
      )}
    </Container>
  )
}

export default GatewayDetails
