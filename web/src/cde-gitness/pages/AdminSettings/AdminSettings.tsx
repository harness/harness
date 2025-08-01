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
  PageError,
  Formik,
  FormikForm
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import * as Yup from 'yup'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useAdminSettings } from './hooks/useAdminSettings'
import GitProviders from './GitProviders/GitProviders'
import CodeEditors from './CodeEditors/CodeEditors'
import CloudRegions from './CloudRegions/CloudRegions'
import GitspaceImages from './GitspaceImages/GitspaceImages'
import css from './AdminSettings.module.scss'

const AdminSettingsPage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()

  const { settings, tabs, initialValues, selectedTab, loading, errorSettings, handleSave, handleTabChange, refetch } =
    useAdminSettings()

  const validationSchema = Yup.object({
    gitspaceImages: Yup.object({
      image_name: Yup.string().when('default_image_added', {
        is: true,
        then: (schema: Yup.StringSchema) =>
          schema.required(getString('validation.nameIsRequired')).trim().min(1, getString('validation.nameIsRequired')),
        otherwise: (schema: Yup.StringSchema) => schema.notRequired()
      })
    }).notRequired()
  })

  return (
    <Formik
      formName="adminSettings"
      initialValues={initialValues}
      onSubmit={handleSave}
      enableReinitialize
      validationSchema={validationSchema}>
      {() => {
        return (
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
                    id={'adminSettingsTabs'}
                    selectedTabId={selectedTab}
                    onChange={handleTabChange}
                    tabList={tabs.map(tab => ({
                      id: tab.id,
                      title: tab.title,
                      panel: (
                        <>
                          {tab.id === 'gitProviders' && <GitProviders />}
                          {tab.id === 'codeEditors' && <CodeEditors />}
                          {tab.id === 'cloudRegions' && <CloudRegions settings={settings} />}
                          {tab.id === 'gitspaceImages' && <GitspaceImages settings={settings} />}
                        </>
                      )
                    }))}
                  />
                </Container>
              )}
            </Page.Body>
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default AdminSettingsPage
