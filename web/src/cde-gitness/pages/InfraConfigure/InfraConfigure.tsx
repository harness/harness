import React, { useEffect, useState } from 'react'
import { Breadcrumbs, Container, Heading, Layout, Page, Tabs, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useParams } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import AWSIcon from 'cde-gitness/assests/aws.svg?url'
import AwsInfraDetails from './AWS/InfraDetails/AwsInfraDetails'
import DownloadAndApplySection from './DownloadAndApply/DownloadAndApply'
import GCPIcon from '../../../icons/google-cloud.svg?url'
import GcpInfraDetails from './InfraDetails/GcpInfraDetails'
import css from './InfraConfigure.module.scss'

const InfraConfigurePage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const { infraprovider_identifier, provider } = useParams<{ infraprovider_identifier?: string; provider: string }>()
  const { type } = useParams<{ type?: string }>()

  const tabOptions = {
    tab1: 'infra-details',
    tab2: 'download-and-apply'
  }
  const [tab, selectedTab] = useState(tabOptions.tab1)

  useEffect(() => {
    selectedTab(type ? tabOptions.tab2 : tabOptions.tab1)
  }, [type])

  return (
    <>
      <Page.Header
        title={
          <Layout.Horizontal>
            <img src={provider === 'hybrid_vm_aws' ? AWSIcon : GCPIcon} width={26} />
            <Heading level={4} font={{ variation: FontVariation.H4 }} padding={{ left: 'small' }}>
              {provider === 'hybrid_vm_aws' ? getString('cde.configureAWSInfra') : getString('cde.configureGCPInfra')}
            </Heading>
          </Layout.Horizontal>
        }
        content={
          <Text
            rightIcon="share"
            color={Color.PRIMARY_7}
            rightIconProps={{ size: 12, color: Color.PRIMARY_7 }}
            className={css.learnMoreLink}
            onClick={() => {
              window.open('/', '_blank')
            }}>
            {getString('cde.configureInfra.learnMoreAboutHybrid')}
          </Text>
        }
        breadcrumbs={
          <Breadcrumbs
            className={css.customBreadcumbStyles}
            links={[
              {
                url: routes.toModuleRoute({ accountId: accountInfo?.identifier }),
                label: `${getString('cde.account')}: ${accountInfo?.name}`
              },
              {
                url: routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier }),
                label: getString('cde.gitspaceInfra')
              },
              {
                url: routes.toCDEInfraConfigureDetail({
                  accountId: accountInfo?.identifier,
                  infraprovider_identifier: infraprovider_identifier ?? '',
                  provider: provider
                }),
                label: getString('cde.configureInfra.configure')
              }
            ]}
          />
        }
      />
      <Page.Body>
        <Container className={css.tabContainer}>
          <Tabs
            id={'horizontalTabs'}
            selectedTabId={tab}
            tabList={[
              {
                id: tabOptions.tab1,
                title: getString('cde.configureInfra.provideInfraDetails'),
                disabled: tab !== tabOptions.tab1,
                panel: provider === 'hybrid_vm_aws' ? <AwsInfraDetails /> : <GcpInfraDetails />
              },
              { id: '', title: <Icon name="chevron-right" size={16} />, disabled: true },
              {
                id: tabOptions.tab2,
                title: getString('cde.configureInfra.applyYamlAndVerifyConnection'),
                disabled: tab !== tabOptions.tab2,
                panel: <DownloadAndApplySection />
              }
            ]}
          />
        </Container>
      </Page.Body>
    </>
  )
}

export default InfraConfigurePage
