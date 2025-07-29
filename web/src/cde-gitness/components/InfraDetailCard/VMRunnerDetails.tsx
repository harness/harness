import React from 'react'
import { Layout, Text, Container } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { HYBRID_VM_GCP } from 'cde-gitness/constants'
import type { TypesInfraProviderConfig } from 'services/cde'
import CopyToClipboard from '../../../components/CopyToClipBoard/CopyToClipBoard'
import { getTruncatedValue } from '../../utils/helper.utils'
import css from './InfraDetailCard.module.scss'

interface VMRunnerDetailsProps {
  infraDetails: TypesInfraProviderConfig
  provider: string
  initialOpen?: boolean
}

const VMRunnerDetails: React.FC<VMRunnerDetailsProps> = ({ infraDetails, provider, initialOpen = true }) => {
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
            {getString('cde.configureInfra.vmRunnerDetails')}
          </Text>
        </Layout.Horizontal>
      </Container>

      {isOpen && (
        <Container className={css.collapsibleContent}>
          <Container className={css.detailsGrid}>
            <Container className={css.detailsGridItem}>
              <Text className={css.rowHeader}>Region</Text>
              <Text
                className={css.rowContent}
                color={Color.BLACK}
                icon="globe-network"
                iconProps={{ size: 16, margin: { right: 'small' } }}>
                {provider === HYBRID_VM_GCP
                  ? infraDetails?.metadata?.runner?.region || ''
                  : infraDetails?.metadata?.runner?.region || ''}
              </Text>
            </Container>
            <Container className={css.detailsGridItem}>
              <Text className={css.rowHeader}>Zone</Text>
              <Text className={css.rowContent}>
                {provider === HYBRID_VM_GCP
                  ? infraDetails?.metadata?.runner?.zone || ''
                  : infraDetails?.metadata?.runner?.availability_zones || ''}
              </Text>
            </Container>
            <Container className={css.detailsGridItem}>
              <Text className={css.rowHeader}>
                {provider === HYBRID_VM_GCP
                  ? getString('cde.gitspaceInfraHome.machineImageName')
                  : getString('cde.Aws.runnerAmiId')}
              </Text>
              <Layout.Horizontal className={css.imageNameContainer} spacing="small">
                <Text
                  className={`${css.rowContent} ${css.truncateText}`}
                  tooltip={
                    provider === HYBRID_VM_GCP
                      ? infraDetails?.metadata?.runner?.vm_image_name || ''
                      : infraDetails?.metadata?.runner?.ami_id || ''
                  }>
                  {getTruncatedValue(
                    provider === HYBRID_VM_GCP
                      ? infraDetails?.metadata?.runner?.vm_image_name || ''
                      : infraDetails?.metadata?.runner?.ami_id || ''
                  )}
                </Text>
                <Container className={css.copyButtonContainer}>
                  <CopyToClipboard
                    content={
                      provider === HYBRID_VM_GCP
                        ? infraDetails?.metadata?.runner?.vm_image_name || ''
                        : infraDetails?.metadata?.runner?.ami_id || ''
                    }
                    iconSize={16}
                    className={css.copyButton}
                  />
                </Container>
              </Layout.Horizontal>
            </Container>
          </Container>
        </Container>
      )}
    </Container>
  )
}

export default VMRunnerDetails
