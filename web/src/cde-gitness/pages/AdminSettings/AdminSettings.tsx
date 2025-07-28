import React from 'react'
import {
  Breadcrumbs,
  Container,
  Heading,
  Layout,
  Page,
  Tabs,
  Button,
  ButtonVariation,
  PageError
} from '@harnessio/uicore'
import { Formik, Form } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useAdminSettings } from './hooks/useAdminSettings'
import GitProviders from './GitProviders/GitProviders'
import CodeEditors from './CodeEditors/CodeEditors'
import CloudRegions from './CloudRegions/CloudRegions'
import css from './AdminSettings.module.scss'

const AdminSettingsPage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()

  const {
    settings,
    tabs,
    initialValues,
    selectedTab,
    loadingSettings,
    errorSettings,
    handleSave,
    handleTabChange,
    refetch,
    loadingUpsert
  } = useAdminSettings()

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
          <Page.Body loading={loadingUpsert || loadingSettings}>
            {errorSettings ? (
              <PageError message={errorSettings.message} onClick={() => refetch()} />
            ) : (
              <Container className={css.tabContainer}>
                <Tabs
                  id={'adminSettingsTabs'}
                  selectedTabId={selectedTab}
                  onChange={handleTabChange}
                  tabList={tabs.map(tab => ({
                    id: tab.id,
                    title: tab.title,
                    panel: (
                      <>
                        {tab.id === 'gitProviders' && <GitProviders settings={settings} />}
                        {tab.id === 'codeEditors' && <CodeEditors settings={settings} />}
                        {tab.id === 'cloudRegions' && <CloudRegions settings={settings} />}
                      </>
                    )
                  }))}
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
