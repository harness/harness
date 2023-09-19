import React, { useState } from 'react'
import { Intent } from '@blueprintjs/core'
import * as yup from 'yup'
import { Color } from '@harnessio/design-system'
import { Button, Container, Layout, FlexExpander, Formik, FormikForm, FormInput, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { REGEX_VALID_REPO_NAME } from 'utils/Utils'
import { ImportFormData, RepoVisibility, parseUrl } from 'utils/GitUtils'
import Private from '../../../icons/private.svg'
import css from '../NewRepoModalButton.module.scss'

interface ImportFormProps {
  handleSubmit: (data: ImportFormData) => void
  loading: boolean // eslint-disable-next-line @typescript-eslint/no-explicit-any
  hideModal: any
}

const ImportForm = (props: ImportFormProps) => {
  const { handleSubmit, loading, hideModal } = props
  const { getString } = useStrings()
  const [auth, setAuth] = useState(false)

  // eslint-disable-next-line no-control-regex
  const MATCH_REPOURL_REGEX = /^(https?:\/\/(?:www\.)?(github|gitlab)\.com\/([^/]+\/[^/]+))/

  const formInitialValues: ImportFormData = {
    repoUrl: '',
    username: '',
    password: '',
    name: '',
    description: '',
    isPublic: RepoVisibility.PRIVATE
  }
  const validationSchemaStepOne = yup.object().shape({
    repoUrl: yup
      .string()
      .matches(MATCH_REPOURL_REGEX, getString('importSpace.invalidUrl'))
      .required(getString('importRepo.required')),
    name: yup
      .string()
      .trim()
      .required(getString('validation.nameIsRequired'))
      .matches(REGEX_VALID_REPO_NAME, getString('validation.repoNamePatternIsNotValid'))
  })
  return (
    <Formik onSubmit={handleSubmit} initialValues={formInitialValues} formName="importRepoForm">
      {formik => {
        const handleValidationClick = async () => {
          try {
            await validationSchemaStepOne.validate(formik.values, { abortEarly: false })
            formik.submitForm()
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
          } catch (err: any) {
            formik.setErrors(
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              err.inner.reduce((acc: { [x: string]: any }, current: { path: string | number; message: string }) => {
                acc[current.path] = current.message
                return acc
              }, {})
            )
          }
        }
        return (
          <FormikForm>
            <FormInput.Text
              className={css.hideContainer}
              name="repoUrl"
              label={getString('importRepo.url')}
              placeholder={getString('importRepo.urlPlaceholder')}
              tooltipProps={{
                dataTooltipId: 'repositoryURLTextField'
              }}
              onChange={event => {
                const target = event.target as HTMLInputElement
                formik.setFieldValue('repoUrl', target.value)
                if (target.value) {
                  const provider = parseUrl(target.value)
                  if (provider?.fullRepo) {
                    formik.setFieldValue('name', provider.repoName ? provider.repoName : provider?.fullRepo)
                    formik.validateField('repoUrl')
                  }
                }
              }}
            />
            {formik.errors.repoUrl ? (
              <Text
                margin={{ top: 'small', bottom: 'small' }}
                color={Color.RED_500}
                icon="circle-cross"
                iconProps={{ color: Color.RED_500 }}>
                {formik.errors.repoUrl}
              </Text>
            ) : null}
            <FormInput.CheckBox
              name="authorization"
              label={getString('importRepo.reqAuth')}
              tooltipProps={{
                dataTooltipId: 'authorization'
              }}
              onClick={() => {
                setAuth(!auth)
              }}
            />

            {auth ? (
              <>
                <FormInput.Text
                  name="username"
                  label={getString('userName')}
                  placeholder={getString('importRepo.userPlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'repositoryUserTextField'
                  }}
                />
                <FormInput.Text
                  inputGroup={{ type: 'password' }}
                  name="password"
                  label={getString('importRepo.passToken')}
                  placeholder={getString('importRepo.passwordPlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'repositoryPasswordTextField'
                  }}
                />
              </>
            ) : null}
            <hr className={css.dividerContainer} />
            <FormInput.Text
              name="name"
              className={css.hideContainer}
              label={getString('name')}
              placeholder={getString('enterRepoName')}
              tooltipProps={{
                dataTooltipId: 'repositoryNameTextField'
              }}
              onChange={() => {
                formik.validateField('repoUrl')
              }}
            />
            {formik.errors.name ? (
              <Text
                margin={{ top: 'small', bottom: 'small' }}
                color={Color.RED_500}
                icon="circle-cross"
                iconProps={{ color: Color.RED_500 }}>
                {formik.errors.name}
              </Text>
            ) : null}
            <FormInput.Text
              name="description"
              label={getString('description')}
              placeholder={getString('enterDescription')}
              tooltipProps={{
                dataTooltipId: 'repositoryDescriptionTextField'
              }}
            />

            <hr className={css.dividerContainer} />

            <Container>
              <FormInput.RadioGroup
                name="isPublic"
                label=""
                items={[
                  {
                    label: (
                      <Container>
                        <Layout.Horizontal>
                          <Icon name="git-clone-step" size={20} margin={{ right: 'medium' }} />
                          <Container>
                            <Layout.Vertical spacing="xsmall">
                              <Text>{getString('public')}</Text>
                              <Text font={{ variation: FontVariation.TINY }}>
                                {getString('createRepoModal.publicLabel')}
                              </Text>
                            </Layout.Vertical>
                          </Container>
                        </Layout.Horizontal>
                      </Container>
                    ),
                    value: RepoVisibility.PUBLIC
                  },
                  {
                    label: (
                      <Container>
                        <Layout.Horizontal>
                          <Container margin={{ right: 'medium' }}>
                            <img width={20} height={20} src={Private} />
                          </Container>
                          {/* <Icon name="git-clone-step" size={20} margin={{ right: 'medium' }} /> */}
                          <Container margin={{ left: 'small' }}>
                            <Layout.Vertical spacing="xsmall">
                              <Text>{getString('private')}</Text>
                              <Text font={{ variation: FontVariation.TINY }}>
                                {getString('createRepoModal.privateLabel')}
                              </Text>
                            </Layout.Vertical>
                          </Container>
                        </Layout.Horizontal>
                      </Container>
                    ),
                    value: RepoVisibility.PRIVATE
                  }
                ]}
              />
            </Container>

            <Layout.Horizontal
              spacing="small"
              padding={{ right: 'xxlarge', top: 'xlarge', bottom: 'large' }}
              style={{ alignItems: 'center' }}>
              <Button
                text={getString('importRepo.title')}
                intent={Intent.PRIMARY}
                disabled={loading}
                onClick={() => {
                  handleValidationClick()
                }}
              />
              <Button text={getString('cancel')} minimal onClick={hideModal} />
              <FlexExpander />

              {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
            </Layout.Horizontal>
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default ImportForm
