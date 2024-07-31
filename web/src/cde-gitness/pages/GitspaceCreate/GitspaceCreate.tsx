import React, { useState } from 'react'
import {
  Breadcrumbs,
  Button,
  ButtonVariation,
  Card,
  Container,
  Formik,
  FormikForm,
  Heading,
  Layout,
  Page,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { ThirdPartyRepoImportForm } from 'cde-gitness/components/ThirdPartyRepoImportForm/ThirdPartyRepoImportForm'
import { GitnessRepoImportForm } from 'cde-gitness/components/GitnessRepoImportForm/GitnessRepoImportForm'
import { SelectIDE } from 'cde/components/CreateGitspace/components/SelectIDE/SelectIDE'
import { useCreateGitspace, type OpenapiCreateGitspaceRequest } from 'cde-gitness/services'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import RepositoryTypeButton, { RepositoryType } from '../../components/RepositoryTypeButton/RepositoryTypeButton'
import { gitnessFormInitialValues } from './GitspaceCreate.constants'
import { validateGitnessForm } from './GitspaceCreate.utils'
import css from './GitspaceCreate.module.scss'

export const GitspaceCreate = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()
  const history = useHistory()
  const [activeButton, setActiveButton] = useState(RepositoryType.GITNESS)
  const { showSuccess, showError } = useToaster()
  const { mutate } = useCreateGitspace({})

  return (
    <>
      <Page.Header
        title={getString('cde.createGitspace')}
        breadcrumbs={
          <Breadcrumbs
            links={[
              { url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') },
              { url: routes.toCDEGitspacesCreate({ space }), label: getString('cde.createGitspace') }
            ]}
          />
        }
      />
      <Page.Body className={css.main}>
        <Container className={css.titleContainer}>
          <Layout.Vertical spacing="small" margin={{ bottom: 'medium' }}>
            <Heading font={{ weight: 'bold' }} color={Color.BLACK} level={2}>
              {getString('cde.createGitspace')}
            </Heading>
            <Text font={{ size: 'medium' }}>{getString('cde.create.subtext')}</Text>
          </Layout.Vertical>
        </Container>
        <Card className={css.cardMain}>
          <Container className={css.subContainers}>
            <Formik
              onSubmit={async data => {
                try {
                  const payload = data
                  const response = await mutate({
                    ...payload,
                    space_ref: space
                  } as OpenapiCreateGitspaceRequest & {
                    space_ref?: string
                  })
                  showSuccess(getString('cde.create.gitspaceCreateSuccess'))
                  history.push(
                    `${routes.toCDEGitspaceDetail({
                      space,
                      gitspaceId: response.identifier || ''
                    })}?redirectFrom=login`
                  )
                } catch (error) {
                  showError(getString('cde.create.gitspaceCreateFailed'))
                  showError(getErrorMessage(error))
                }
              }}
              initialValues={gitnessFormInitialValues}
              validationSchema={validateGitnessForm(getString)}
              formName="importRepoForm"
              enableReinitialize>
              {formik => {
                return (
                  <>
                    <Layout.Horizontal
                      className={css.formTitleContainer}
                      flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
                      <Text font={{ variation: FontVariation.CARD_TITLE }}>
                        {getString('cde.create.repositoryDetails')}
                      </Text>
                      <RepositoryTypeButton
                        hasChange={formik.dirty}
                        onChange={type => {
                          setActiveButton(type)
                          formik?.resetForm()
                        }}
                      />
                    </Layout.Horizontal>
                    <FormikForm>
                      <Container className={css.formContainer}>
                        {activeButton === RepositoryType.GITNESS && (
                          <Container>
                            <GitnessRepoImportForm />
                          </Container>
                        )}
                        {activeButton === RepositoryType.THIRDPARTY && (
                          <Container>
                            <ThirdPartyRepoImportForm />
                          </Container>
                        )}
                      </Container>
                      <Container className={css.formOuterContainer}>
                        <SelectIDE standalone />
                        <Button width={'100%'} variation={ButtonVariation.PRIMARY} height={50} type="submit">
                          {getString('cde.createGitspace')}
                        </Button>
                      </Container>
                    </FormikForm>
                  </>
                )
              }}
            </Formik>
          </Container>
        </Card>
      </Page.Body>
    </>
  )
}
