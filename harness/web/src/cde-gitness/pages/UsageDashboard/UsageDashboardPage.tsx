import React, { useState } from 'react'
import { Breadcrumbs, Container, Page, Tabs } from '@harnessio/uicore'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
//import UsageTabPanel from './UsageTabPanel'
import GitspacesTabPanel from './GitspacesTabPanel'
import css from './UsageDashboardPage.module.scss'

// Tab constants
//const USAGE_TAB = 'usage'
const GITSPACES_TAB = 'gitspaces'

// Interface for query params
interface DashboardQueryParams {
  tab?: string
}

const UsageDashboardPage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const { updateQueryParams } = useUpdateQueryParams()
  const queryParams = useQueryParams<DashboardQueryParams>()

  const [selectedTab, setSelectedTab] = useState(queryParams.tab || GITSPACES_TAB)

  const handleTabChange = (tabId: string) => {
    setSelectedTab(tabId)
    updateQueryParams({ tab: tabId })
  }

  return (
    <>
      <Page.Header
        title={getString('cde.usageDashboard.title')}
        breadcrumbs={
          <Breadcrumbs
            className={css.customBreadcumbStyles}
            links={[
              {
                url: routes.toModuleRoute({ accountId: accountInfo?.identifier }),
                label: `${getString('cde.account')}: ${accountInfo?.name}`
              },
              {
                url: routes.toCDEUsageDashboard({ accountId: accountInfo?.identifier }),
                label: getString('cde.usageDashboard.title')
              }
            ]}
          />
        }
      />
      <Container className={css.tabs}>
        <Tabs
          id="usage-dashboard-tabs"
          selectedTabId={selectedTab}
          onChange={handleTabChange}
          tabList={[
            // {
            //   id: USAGE_TAB,
            //   title: getString('cde.usageDashboard.usageTab'),
            //   panel: <UsageTabPanel />
            // },
            {
              id: GITSPACES_TAB,
              title: getString('cde.usageDashboard.gitspacesTab'),
              panel: <GitspacesTabPanel />
            }
          ]}
        />
      </Container>
    </>
  )
}

export default UsageDashboardPage
