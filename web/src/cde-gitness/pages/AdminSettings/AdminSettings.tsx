import React, { useState } from 'react'
import {
  Breadcrumbs,
  Container,
  Heading,
  Layout,
  Page,
  Tabs,
  Button,
  ButtonVariation,
  useToaster,
  PageSpinner,
  PageError
} from '@harnessio/uicore'
import { Formik, Form } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import {
  useFindGitspaceSettings,
  useUpsertGitspaceSettings,
  TypesGitspaceSettingsData,
  EnumGitspaceCodeRepoType
} from 'services/cde'
import { scmOptionsCDE, SCMType } from 'cde-gitness/pages/GitspaceCreate/CDECreateGitspace'
import { routes } from 'cde-gitness/RouteDefinitions'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import GitProviders from './GitProviders/GitProviders'
import css from './AdminSettings.module.scss'

interface AdminSettingsFormValues {
  gitProviders: {
    [key: string]: boolean
  }
}

const AdminSettingsPage = () => {
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const { accountInfo } = useAppContext()

  const {
    data: settings,
    loading: loadingSettings,
    error: errorSettings,
    refetch
  } = useFindGitspaceSettings({
    accountIdentifier: accountInfo?.identifier
  })

  const { mutate: upsertSettings } = useUpsertGitspaceSettings({
    accountIdentifier: accountInfo?.identifier
  })

  const tabOptions = {
    gitProviders: 'git-providers',
    codeEditors: 'code-editors',
    cloudRegions: 'cloud-regions',
    gitspaceImages: 'gitspace-images'
  }

  const [selectedTab, setSelectedTab] = useState(tabOptions.gitProviders)

  const initialValues: AdminSettingsFormValues = {
    gitProviders: {
      ...scmOptionsCDE.reduce((acc, provider: SCMType) => {
        acc[provider.value] = true
        return acc
      }, {} as { [key: string]: boolean })
    }
  }

  const handleSave = async (values: AdminSettingsFormValues) => {
    const allProviders = scmOptionsCDE.map(p => p.value)
    const deniedProviders = allProviders.filter(
      provider => !values.gitProviders[provider]
    ) as EnumGitspaceCodeRepoType[]

    const payload: TypesGitspaceSettingsData = {
      ...settings?.settings,
      gitspace_config: {
        ...settings?.settings?.gitspace_config,
        scm: {
          access_list: {
            mode: 'deny',
            list: deniedProviders
          }
        }
      }
    }

    try {
      await upsertSettings(payload)
      showSuccess(getString('cde.settings.saveSuccess'))
    } catch (err) {
      showError(getErrorMessage(err))
    }
  }

  return (
    <Formik initialValues={initialValues} onSubmit={handleSave} enableReinitialize>
      {() => (
        <Form>
          <Page.Header
            title={
              <Layout.Horizontal>
                <Heading level={4} font={{ variation: FontVariation.H4 }}>
                  {getString('cde.gitspaces')}
                </Heading>
              </Layout.Horizontal>
            }
            content={
              <Layout.Horizontal spacing="medium">
                <Button type="submit" variation={ButtonVariation.PRIMARY}>
                  {getString('save')}
                </Button>
              </Layout.Horizontal>
            }
            breadcrumbs={
              <Breadcrumbs
                className={css.customBreadcumbStyles}
                links={[
                  {
                    url: routes.toModuleRoute({ accountId: accountInfo?.identifier }),
                    label: `${getString('cde.account')}: ${accountInfo?.name}`
                  }
                ]}
              />
            }
          />
          <Page.Body>
            {loadingSettings ? (
              <PageSpinner />
            ) : errorSettings ? (
              <PageError message={errorSettings.message} onClick={() => refetch()} />
            ) : (
              <Container className={css.tabContainer}>
                <Tabs
                  id={'adminSettingsTabs'}
                  selectedTabId={selectedTab}
                  onChange={(tabId: string) => setSelectedTab(tabId)}
                  tabList={[
                    {
                      id: tabOptions.gitProviders,
                      title: getString('cde.settings.gitProviders'),
                      panel: <GitProviders settings={settings} />
                    },
                    {
                      id: tabOptions.codeEditors,
                      title: getString('cde.settings.codeEditors'),
                      disabled: true,
                      panel: <></>
                    },
                    {
                      id: tabOptions.cloudRegions,
                      title: getString('cde.settings.cloudRegionsAndMachineTypes'),
                      disabled: true,
                      panel: <></>
                    },
                    {
                      id: tabOptions.gitspaceImages,
                      title: getString('cde.settings.gitspaceImages'),
                      disabled: true,
                      panel: <></>
                    }
                  ]}
                />
              </Container>
            )}
          </Page.Body>
        </Form>
      )}
    </Formik>
  )
}

export default AdminSettingsPage
