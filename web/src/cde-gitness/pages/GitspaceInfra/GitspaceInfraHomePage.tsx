import React, { useEffect, useState } from 'react'
import {
  Breadcrumbs,
  Button,
  ButtonVariation,
  Container,
  Layout,
  Page,
  Tabs,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import { HYBRID_VM_GCP, regionType } from 'cde-gitness/constants'
import { useInfraListingApi } from 'cde-gitness/hooks/useGetInfraListProvider'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage } from 'utils/Utils'
import { TypesInfraProviderConfig, useDeleteInfraProvider } from 'services/cde'
import InfraLoaderCard from 'cde-gitness/components/InfraLoaderCard/InfraLoaderCard'
import InfraDetailCard from 'cde-gitness/components/InfraDetailCard/InfraDetailCard'
import NoDataCard from './NoDataCard/NoDataCard'
import MachineLocationContent from './MachineLocationContent/MachineLocationContent'
import css from './GitspaceInfraHomePage.module.scss'

interface TabData {
  id: string
  title: JSX.Element
  panel: JSX.Element
}

const GitspaceInfraHomePage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const history = useHistory()
  const [selectedTab, setSelectedTab] = useState('')
  const [infraDetails, setInfraDetails] = useState<TypesInfraProviderConfig>()
  const [regionData, setRegionData] = useState<regionType[]>()
  const { showError, showSuccess } = useToaster()
  const [isConnected, setIsConnected] = useState(true)

  const { mutate: deleteInfraProvider } = useDeleteInfraProvider({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: infraDetails?.identifier ?? ''
  })

  const [machineLocationTabs, setMachineLocationTabs] = useState<TabData[]>([])
  const { data: listResponse, loading = false, refetch } = useInfraListingApi()

  useEffect(() => {
    refetch()
  }, [])

  useEffect(() => {
    generateTabData()
  }, [regionData])

  useEffect(() => {
    let infra: any = null
    listResponse?.forEach((infraProvider: TypesInfraProviderConfig) => {
      if (infraProvider?.type === HYBRID_VM_GCP) {
        infra = infraProvider
      }
    })
    if (infra) {
      const regions: regionType[] = []
      const region_configs = infra?.metadata?.region_configs ?? {}
      Object.keys(region_configs)?.forEach((key: string) => {
        const { region_name, proxy_subnet_ip_range, default_subnet_ip_range, certificates } = region_configs?.[key]
        regions.push({
          region_name,
          proxy_subnet_ip_range,
          default_subnet_ip_range,
          dns: certificates?.contents?.[0]?.dns_managed_zone_name,
          domain: certificates?.contents?.[0]?.domain,
          machines: infra?.resources?.filter((item: any) => item?.metadata?.region_name === region_name) ?? []
        })
      })
      setIsConnected(true)
      setInfraDetails(infra)
      setRegionData(regions)
    } else {
      setInfraDetails(undefined)
      setRegionData([])
    }
  }, [listResponse])

  const generateTabData = () => {
    const tabList: TabData[] = []
    regionData?.forEach((tab: regionType) => {
      tabList.push({
        id: tab?.region_name,
        title: (
          <Layout.Horizontal spacing={'medium'}>
            <Text
              className={css.tabHeading}
              color={selectedTab === tab?.region_name ? Color.GREY_1000 : Color.GREY_500}>
              {tab?.region_name}
            </Text>
            <Text className={css.countLabel}>{tab?.machines?.length ?? 0}</Text>
          </Layout.Horizontal>
        ),
        panel: (
          <MachineLocationContent
            locationData={tab}
            machineData={tab?.machines}
            isConnected={isConnected}
            infraprovider_identifier={infraDetails?.identifier ?? ''}
            setRegionData={setRegionData}
            regionData={regionData}
          />
        )
      })
    })
    if (selectedTab === '' && tabList?.length > 0) {
      setSelectedTab(tabList[0]?.id)
    }
    setMachineLocationTabs(tabList)
  }

  const handleTabChange = (tabId: string) => {
    setSelectedTab(tabId)
  }

  const confirmDelete = useConfirmAct()

  const handleInfraDelete = (e: React.MouseEvent) => {
    confirmDelete({
      intent: 'danger',
      title: `${getString('cde.gitspaceInfraHome.deleteInfraTitle')}`,
      message: getString('cde.gitspaceInfraHome.deleteInfraText'),
      confirmText: getString('delete'),
      action: async () => {
        try {
          e.preventDefault()
          e.stopPropagation()
          await deleteInfraProvider('')
          showSuccess(getString('cde.deleteInfraSuccess'))
          refetch()
        } catch (exception) {
          showError(getErrorMessage(exception))
        }
      }
    })
  }

  return (
    <>
      <Page.Header
        title={getString('cde.gitspaceInfra')}
        content={
          infraDetails ? (
            <Button
              icon="Edit"
              iconProps={{ size: 12 }}
              variation={ButtonVariation.SECONDARY}
              text={getString('cde.edit')}
              onClick={() =>
                history.push(
                  routes.toCDEInfraConfigureDetail({
                    accountId: accountInfo?.identifier,
                    infraprovider_identifier: infraDetails?.identifier ?? ''
                  })
                )
              }
            />
          ) : (
            <></>
          )
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
              }
            ]}
          />
        }
      />
      {loading ? (
        <InfraLoaderCard />
      ) : (
        <>
          {infraDetails ? (
            <Page.Body className={css.main}>
              <Layout.Vertical spacing={'xlarge'}>
                <InfraDetailCard infraDetails={infraDetails} regionCount={regionData?.length ?? 0} />

                <Container className={css.locationAndMachineCard}>
                  <Layout.Vertical spacing={'none'}>
                    <Text className={css.locationAndMachineTitle} color={Color.GREY_1000}>
                      {getString('cde.gitspaceInfraHome.locationAndMachine')}
                    </Text>
                    <Tabs
                      id={'horizontalTabs'}
                      selectedTabId={selectedTab}
                      tabList={machineLocationTabs}
                      onChange={handleTabChange}
                    />
                  </Layout.Vertical>
                </Container>

                <Layout.Vertical className={css.deleteInfraContainer}>
                  <Text className={css.deleteInfraTitle}>{getString('cde.gitspaceInfraHome.dangerZone')}</Text>
                  <Container className={css.deleteInfraCard}>
                    <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                      <Layout.Vertical>
                        <Text className={css.deleteHeading} color={Color.GREY_1000}>
                          {getString('cde.gitspaceInfraHome.deleteThisInfra')}
                        </Text>
                        <Text className={css.deleteMessage} color={Color.GREY_300}>
                          {getString('cde.gitspaceInfraHome.deleteWarning')}
                        </Text>
                      </Layout.Vertical>
                      <Button
                        text={getString('cde.gitspaceInfraHome.deleteThisInfra')}
                        variation={ButtonVariation.TERTIARY}
                        className={css.deleteBtn}
                        color={Color.RED_600}
                        onClick={handleInfraDelete}
                      />
                    </Layout.Horizontal>
                  </Container>
                </Layout.Vertical>
              </Layout.Vertical>
            </Page.Body>
          ) : (
            <NoDataCard />
          )}
        </>
      )}
    </>
  )
}

export default GitspaceInfraHomePage
