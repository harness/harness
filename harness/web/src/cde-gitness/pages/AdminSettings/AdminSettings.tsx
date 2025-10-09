import React from 'react'
import {
  Page,
  Container,
  Button,
  ButtonVariation,
  Tabs,
  Layout,
  Heading,
  Breadcrumbs,
  PageError,
  Formik,
  FormikForm
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useAdminSettings } from './hooks/useAdminSettings'
import { getValidationSchema, AdminSettingsTabs } from './utils/adminSettingsUtils'
import GitProviders from './GitProviders/GitProviders'
import CodeEditors from './CodeEditors/CodeEditors'
import CloudRegions from './CloudRegions/CloudRegions'
import GitspaceImages from './GitspaceImages/GitspaceImages'
import css from './AdminSettings.module.scss'

const AdminSettingsPage: React.FC = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()

  const { settings, tabs, initialValues, selectedTab, loading, errorSettings, handleSave, handleTabChange, refetch } =
    useAdminSettings()

  return (
    <Formik
      formName="adminSettings"
      initialValues={initialValues}
      onSubmit={handleSave}
      enableReinitialize
      validationSchema={getValidationSchema(getString)}>
      {() => (
        <FormikForm>
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
          <Page.Body loading={loading}>
            {errorSettings ? (
              <PageError message={errorSettings.message} onClick={() => refetch()} />
            ) : (
              <Container className={css.tabContainer}>
                <Tabs
                  id="adminSettingsTabs"
                  selectedTabId={selectedTab}
                  onChange={handleTabChange}
                  tabList={tabs.map(tab => ({
                    id: tab.id,
                    title: tab.title,
                    panel: (
                      <>
                        {tab.id === AdminSettingsTabs.GIT_PROVIDERS && <GitProviders />}
                        {tab.id === AdminSettingsTabs.CODE_EDITORS && <CodeEditors />}
                        {tab.id === AdminSettingsTabs.CLOUD_REGIONS && <CloudRegions settings={settings} />}
                        {tab.id === AdminSettingsTabs.GITSPACE_IMAGES && <GitspaceImages settings={settings} />}
                      </>
                    )
                  }))}
                />
              </Container>
            )}
          </Page.Body>
        </FormikForm>
      )}
    </Formik>
  )
}

export default AdminSettingsPage
