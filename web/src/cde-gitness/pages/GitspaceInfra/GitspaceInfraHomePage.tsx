import React, { useEffect, useState } from 'react'
import { Breadcrumbs, Container, Page, Tabs } from '@harnessio/uicore'
import { useHistory, useLocation } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useInfraListingApi } from 'cde-gitness/hooks/useGetInfraListProvider'
import InfraLoaderCard from 'cde-gitness/components/InfraLoaderCard/InfraLoaderCard'
import { routes } from 'cde-gitness/RouteDefinitions'
import { HYBRID_VM_AWS, HYBRID_VM_GCP } from 'cde-gitness/constants'
import GcpInfrastructurePanel from './panels/GcpInfrastructurePanel'
import AwsInfrastructurePanel from './panels/AwsInfrastructurePanel'
import css from './GitspaceInfraHomePage.module.scss'

const GitspaceInfraHomePage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const history = useHistory()
  const location = useLocation()
  const searchParam = new URLSearchParams(location.search)
  const initialInfraType = searchParam.get('type')
  const [selectedInfraTab, setSelectedInfraTab] = useState(
    initialInfraType === HYBRID_VM_AWS ? HYBRID_VM_AWS : HYBRID_VM_GCP
  )

  const {
    data: listResponse,
    loading = false,
    refetch
  } = useInfraListingApi({
    queryParams: {
      acl_filter: 'false'
    }
  })

  useEffect(() => {
    refetch()
  }, [])

  useEffect(() => {
    // Handle query parameters for tab switching
    const searchParams = new URLSearchParams(location.search)
    const infraType = searchParams.get('type')
    if (infraType === HYBRID_VM_AWS) {
      setSelectedInfraTab(HYBRID_VM_AWS)
    } else {
      setSelectedInfraTab(HYBRID_VM_GCP)
    }
  }, [location.search])

  const handleInfraTabChange = (tabId: string) => {
    setSelectedInfraTab(tabId as typeof HYBRID_VM_AWS | typeof HYBRID_VM_GCP)
    // Update URL query parameter
    const searchParams = new URLSearchParams(location.search)
    if (tabId === HYBRID_VM_AWS) {
      searchParams.set('type', HYBRID_VM_AWS)
    } else {
      searchParams.delete('type')
    }
    const newSearch = searchParams.toString()
    history.replace({
      pathname: location.pathname,
      search: newSearch ? `?${newSearch}` : ''
    })
  }

  return (
    <>
      <Page.Header
        title={getString('cde.gitspaceInfra')}
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
              }
            ]}
          />
        }
      />
      {loading ? (
        <InfraLoaderCard />
      ) : (
        <Container className={css.tabs}>
          <Tabs
            id={'infraTabs'}
            selectedTabId={selectedInfraTab}
            tabList={[
              {
                id: HYBRID_VM_GCP,
                title: getString('cde.gcpInfrastructure'),
                panel: <GcpInfrastructurePanel listResponse={listResponse} loading={loading} refetch={refetch} />
              },
              {
                id: HYBRID_VM_AWS,
                title: getString('cde.awsInfrastructure'),
                panel: <AwsInfrastructurePanel listResponse={listResponse} loading={loading} refetch={refetch} />
              }
            ]}
            onChange={handleInfraTabChange}
          />
        </Container>
      )}
    </>
  )
}

export default GitspaceInfraHomePage
