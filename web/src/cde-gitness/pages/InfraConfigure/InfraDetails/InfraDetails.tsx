import React from 'react'
import { Button, ButtonVariation, Container, Formik, FormikForm, Layout, Page } from '@harnessio/uicore'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useAppContext } from 'AppContext'
import BasicDetails from './BasicDetails'
import GatewayDetails from './GatewayDetails'
import ConfigureLocations from './ConfigureLocations'
import css from './InfraDetails.module.scss'

interface InfraDetailProps {
  onTabChange: (key: string) => void
  tabOptions: { [key: string]: string }
}

const InfraDetails = ({ onTabChange, tabOptions }: InfraDetailProps) => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const history = useHistory()
  return (
    <Page.Body className={css.main}>
      <Container className={css.basicDetailsContainer}>
        <Formik
          formName="edit-layout-name"
          onSubmit={() => {
            // handleSubmit(values)
          }}
          initialValues={{}}
          enableReinitialize>
          {() => {
            return (
              <FormikForm>
                <Layout.Vertical spacing="medium">
                  <BasicDetails />
                  <GatewayDetails />
                  <ConfigureLocations />
                  <Layout.Horizontal className={css.formFooter}>
                    <Button
                      text={getString('cde.configureInfra.cancel')}
                      variation={ButtonVariation.TERTIARY}
                      onClick={() => history.push(routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier }))}
                    />
                    <Button
                      text={getString('cde.configureInfra.downloadAndApply')}
                      rightIcon="chevron-right"
                      variation={ButtonVariation.PRIMARY}
                      onClick={() => onTabChange(tabOptions.tab2)}
                    />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </FormikForm>
            )
          }}
        </Formik>
      </Container>
    </Page.Body>
  )
}

export default InfraDetails
