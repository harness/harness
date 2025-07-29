import React from 'react'
import { Layout, Text } from '@harnessio/uicore'
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
    <div className={css.collapsibleSection}>
      <div className={css.collapsibleHeader} onClick={() => setIsOpen(!isOpen)}>
        <Layout.Horizontal spacing="small" flex>
          <Text
            className={css.sectionTitle}
            color={Color.BLACK}
            icon={isOpen ? 'chevron-down' : 'chevron-right'}
            iconProps={{ size: 16, margin: { right: 'small' } }}>
            {getString('cde.configureInfra.vmRunnerDetails')}
          </Text>
        </Layout.Horizontal>
      </div>

      {isOpen && (
        <div className={css.collapsibleContent}>
          <div className={css.detailsGrid}>
            <div className={css.detailsGridItem}>
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
            </div>
            <div className={css.detailsGridItem}>
              <Text className={css.rowHeader}>Zone</Text>
              <Text className={css.rowContent}>
                {provider === HYBRID_VM_GCP
                  ? infraDetails?.metadata?.runner?.zone || ''
                  : infraDetails?.metadata?.runner?.availability_zones || ''}
              </Text>
            </div>
            <div className={css.detailsGridItem}>
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
                <div className={css.copyButtonContainer}>
                  <CopyToClipboard
                    content={
                      provider === HYBRID_VM_GCP
                        ? infraDetails?.metadata?.runner?.vm_image_name || ''
                        : infraDetails?.metadata?.runner?.ami_id || ''
                    }
                    iconSize={16}
                    className={css.copyButton}
                  />
                </div>
              </Layout.Horizontal>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default VMRunnerDetails
