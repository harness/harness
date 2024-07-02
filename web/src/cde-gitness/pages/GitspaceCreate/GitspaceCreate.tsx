import React, { useState } from 'react'
import {
  Button,
  ButtonVariation,
  Card,
  Container,
  Formik,
  FormikForm,
  Layout,
  Page,
  Text,
  useToaster
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import {
  ThirdPartyRepoImportForm,
  ThirdPartyRepoImportFormProps
} from 'cde-gitness/components/ThirdPartyRepoImportForm/ThirdPartyRepoImportForm'
import { GitnessRepoImportForm } from 'cde-gitness/components/GitnessRepoImportForm/GitnessRepoImportForm'
import { SelectIDE } from 'cde/components/CreateGitspace/components/SelectIDE/SelectIDE'
import { useCreateGitspace, type OpenapiCreateGitspaceRequest } from 'cde-gitness/services'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import RepositoryTypeButton, { RepositoryType } from '../../components/RepositoryTypeButton/RepositoryTypeButton'
import { gitnessFormInitialValues, thirdPartyformInitialValues } from './GitspaceCreate.constants'
import { handleImportSubmit, validateGitnessForm, validationSchemaStepOne } from './GitspaceCreate.utils'
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
      <Page.Header title={getString('cde.gitspaces')} />
      <Page.Body className={css.main}>
        <Card className={css.cardMain}>
          <Text className={css.cardTitle} font={{ variation: FontVariation.CARD_TITLE }}>
            {getString('cde.createGitspace')}
          </Text>
          <Container className={css.subContainers}>
            <Formik
              onSubmit={async data => {
                try {
                  const payload =
                    activeButton === RepositoryType.GITNESS
                      ? data
                      : handleImportSubmit(data as ThirdPartyRepoImportFormProps)
                  await mutate({ ...payload, space_ref: space } as OpenapiCreateGitspaceRequest & {
                    space_ref?: string
                  })
                  showSuccess(getString('cde.create.gitspaceCreateSuccess'))
                  history.push(routes.toCDEGitspaces({ space }))
                } catch (error) {
                  showError(getString('cde.create.gitspaceCreateFailed'))
                  showError(getErrorMessage(error))
                }
              }}
              initialValues={
                activeButton === RepositoryType.GITNESS ? gitnessFormInitialValues : thirdPartyformInitialValues
              }
              validationSchema={
                activeButton === RepositoryType.GITNESS
                  ? validateGitnessForm(getString)
                  : validationSchemaStepOne(getString)
              }
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
                        <SelectIDE />
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
