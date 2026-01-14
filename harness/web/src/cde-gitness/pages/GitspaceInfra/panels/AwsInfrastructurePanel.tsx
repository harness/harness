import React, { useEffect, useState } from 'react'
import { Button, ButtonVariation, Container, Layout, Page, Tabs, Text, useToaster } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { HYBRID_VM_AWS, regionType } from 'cde-gitness/constants'
import type { TypesInfraProviderConfig } from 'services/cde'
import { useAwsInfrastructure } from 'cde-gitness/hooks/useAwsInfrastructure'
import { useDeleteInfraProvider } from 'services/cde'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage } from 'utils/Utils'
import { downloadYaml } from 'cde-gitness/utils/helper.utils'
import { routes } from 'cde-gitness/RouteDefinitions'
import InfraDetailCard from 'cde-gitness/components/InfraDetailCard/InfraDetailCard'
import { useGetInfraDetails } from 'cde-gitness/hooks/useInfraDetailAPI'
import AWSIcon from 'cde-gitness/assests/aws.svg?url'
import NoDataCard from '../NoDataCard/NoDataCard'
import MachineLocationContent from '../MachineLocationContent/MachineLocationContent'
import css from '../GitspaceInfraHomePage.module.scss'

interface TabData {
  id: string
  title: JSX.Element
  panel: JSX.Element
}

interface AwsInfrastructurePanelProps {
  listResponse?: TypesInfraProviderConfig[] | null
  loading?: boolean
  refetch: () => void
}

const AwsInfrastructurePanel: React.FC<AwsInfrastructurePanelProps> = ({ listResponse, loading, refetch }) => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const history = useHistory()
  const { showError, showSuccess } = useToaster()
  const [selectedTab, setSelectedTab] = useState('')
  const [machineLocationTabs, setMachineLocationTabs] = useState<TabData[]>([])

  const { awsInfraDetails, awsRegionData, setAwsRegionData, isConnected } = useAwsInfrastructure({
    listResponse,
    loading,
    refetch
  })

  const { mutate: deleteInfraProvider } = useDeleteInfraProvider({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: awsInfraDetails?.identifier ?? ''
  })

  const { data: infraDetailsData, loading: infraDetailsLoading } = useGetInfraDetails({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: awsInfraDetails?.identifier ?? '',
    queryParams: {}
  })

  const confirmDelete = useConfirmAct()

  useEffect(() => {
    generateTabData()
  }, [awsRegionData])

  const generateTabData = () => {
    const tabList: TabData[] = []
    awsRegionData?.forEach((tab: regionType) => {
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
            infraprovider_identifier={awsInfraDetails?.identifier ?? ''}
            setRegionData={setAwsRegionData}
            regionData={awsRegionData}
            provider={HYBRID_VM_AWS}
            infraDetails={awsInfraDetails}
            refetch={refetch}
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
          refetch?.()
        } catch (exception) {
          showError(getErrorMessage(exception))
        }
      }
    })
  }

  if (!awsInfraDetails) {
    return <NoDataCard provider={HYBRID_VM_AWS} />
  }

  const handleDownloadYaml = () => {
    downloadYaml(infraDetailsData?.setup_yaml, 'aws-infra-config.yaml', () =>
      showError(getString('cde.configureInfra.yamlNotAvailable'))
    )
  }

  return (
    <Page.Body className={css.main}>
      <Layout.Vertical spacing={'xlarge'}>
        <Layout.Horizontal flex={{ justifyContent: 'space-between' }} spacing={'normal'}>
          <Layout.Horizontal spacing={'normal'}>
            <img src={AWSIcon} width={24} className={css.infraTitle} />
            <Layout.Vertical>
              <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
                {getString('cde.awsInfrastructureName')}
              </Text>
              <Text font={{ variation: FontVariation.H4 }}>{awsInfraDetails?.name}</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
          <Layout.Horizontal spacing={'medium'}>
            <Button
              icon="Edit"
              iconProps={{ size: 12 }}
              variation={ButtonVariation.SECONDARY}
              text={getString('cde.edit')}
              onClick={() =>
                history.push(
                  routes.toCDEInfraConfigureDetail({
                    accountId: accountInfo?.identifier,
                    infraprovider_identifier: awsInfraDetails?.identifier ?? '',
                    provider: HYBRID_VM_AWS
                  })
                )
              }
            />
            <Button
              icon="download-manifests-inverse"
              iconProps={{ size: 14 }}
              variation={ButtonVariation.PRIMARY}
              text={
                infraDetailsLoading
                  ? getString('cde.configureInfra.yamlLoading')
                  : getString('cde.configureInfra.downloadInfraYaml')
              }
              onClick={handleDownloadYaml}
              className={css.downloadButton}
              disabled={infraDetailsLoading || !infraDetailsData?.setup_yaml}
              tooltip={
                infraDetailsLoading || !infraDetailsData?.setup_yaml
                  ? getString('cde.configureInfra.yamlNotAvailable')
                  : ''
              }
            />
          </Layout.Horizontal>
        </Layout.Horizontal>
        <InfraDetailCard
          provider={HYBRID_VM_AWS}
          infraDetails={awsInfraDetails}
          regionCount={awsRegionData?.length ?? 0}
        />

        <Container className={css.locationAndMachineCard}>
          <Layout.Vertical spacing={'none'}>
            <Text className={css.locationAndMachineTitle} color={Color.GREY_1000}>
              {getString('cde.gitspaceInfraHome.regionsAndMachines')}
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
  )
}

export default AwsInfrastructurePanel
