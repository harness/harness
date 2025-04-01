import React, { useEffect, useState } from 'react'
import { Breadcrumbs, Container, Heading, Layout, Page, Tabs, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useParams } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import InfraDetails from './InfraDetails/InfraDetails'
import DownloadAndApplySection from './DownloadAndApply/DownloadAndApply'
import GCPIcon from '../../../icons/google-cloud.svg?url'
import css from './InfraConfigure.module.scss'

const InfraConfigurePage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
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
            <img src={GCPIcon} width={24} />
            <Heading level={4} font={{ variation: FontVariation.H4 }} padding={{ left: 'small' }}>
              {getString('cde.configureGCPInfra')}
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
                url: routes.toCDEInfraConfigure({ accountId: accountInfo?.identifier }),
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
                panel: <InfraDetails />
              },
              { id: '', title: <Icon name="chevron-right" size={16} />, disabled: true },
              {
                id: tabOptions.tab2,
                title: getString('cde.configureInfra.downloadAndApply'),
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
