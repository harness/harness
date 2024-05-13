/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState } from 'react'
import { Intent } from '@blueprintjs/core'
import * as yup from 'yup'
import { Color } from '@harnessio/design-system'
import { Button, Container, Label, Layout, FlexExpander, Formik, FormikForm, FormInput, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'
import { type ImportSpaceFormData, GitProviders, getProviders, getOrgLabel, getOrgPlaceholder } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import css from '../../NewSpaceModalButton/NewSpaceModalButton.module.scss'

interface ImportReposProps {
  handleSubmit: (data: ImportSpaceFormData) => void
  loading: boolean
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  hideModal: any
  spaceRef: string
}

const getHostPlaceHolder = (gitProvider: string) => {
  switch (gitProvider) {
    case GitProviders.GITHUB:
    case GitProviders.GITHUB_ENTERPRISE:
      return 'enterGithubPlaceholder'
    case GitProviders.GITLAB:
    case GitProviders.GITLAB_SELF_HOSTED:
      return 'enterGitlabPlaceholder'
    case GitProviders.BITBUCKET:
    case GitProviders.BITBUCKET_SERVER:
      return 'enterBitbucketPlaceholder'
    default:
      return 'enterAddress'
  }
}

const ImportReposForm = (props: ImportReposProps) => {
  const { handleSubmit, loading, hideModal, spaceRef } = props
  const { standalone } = useAppContext()
  const { getString } = useStrings()
  const [auth, setAuth] = useState(false)
  const [step, setStep] = useState(0)
  const [buttonLoading, setButtonLoading] = useState(false)

  const formInitialValues: ImportSpaceFormData = {
    gitProvider: GitProviders.GITHUB,
    username: '',
    password: '',
    name: spaceRef,
    description: '',
    organization: '',
    project: '',
    host: '',
    importPipelineLabel: false
  }

  const validationSchemaStepOne = yup.object().shape({
    gitProvider: yup.string().trim().required(getString('importSpace.providerRequired'))
  })

  const validationSchemaStepTwo = yup.object().shape({
    organization: yup.string().trim().required(getString('importSpace.orgRequired')),
    project: yup
      .string()
      .trim()
      .when('gitProvider', {
        is: GitProviders.AZURE,
        then: yup.string().required(getString('importSpace.spaceNameRequired'))
      })
  })

  return (
    <Formik
      initialValues={formInitialValues}
      formName="importReposForm"
      enableReinitialize={true}
      validateOnBlur
      onSubmit={handleSubmit}>
      {formik => {
        const { values } = formik
        const handleValidationClick = async () => {
          try {
            if (step === 0) {
              await validationSchemaStepOne.validate(formik.values, { abortEarly: false })
              setStep(1)
            } else if (step === 1) {
              await validationSchemaStepTwo.validate(formik.values, { abortEarly: false })
              setButtonLoading(true)
            } // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
        const handleImport = async () => {
          await handleSubmit(formik.values)
          setButtonLoading(false)
        }

        return (
          <Container className={css.hideContainer} width={'97%'}>
            <FormikForm>
              {step === 0 ? (
                <>
                  <Container width={'70%'}>
                    <Layout.Horizontal>
                      <Text padding={{ left: 'small' }} font={{ size: 'small' }}>
                        {getString('importRepos.content')}
                      </Text>
                    </Layout.Horizontal>
                  </Container>
                  <hr className={css.dividerContainer} />
                  <Container className={css.textContainer} width={'70%'}>
                    <FormInput.Select
                      name={'gitProvider'}
                      label={getString('importSpace.gitProvider')}
                      items={getProviders()}
                      className={css.selectBox}
                    />
                    {formik.errors.gitProvider ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.gitProvider}
                      </Text>
                    ) : null}
                    {![GitProviders.GITHUB, GitProviders.GITLAB, GitProviders.BITBUCKET, GitProviders.AZURE].includes(
                      values.gitProvider
                    ) && (
                      <FormInput.Text
                        name="host"
                        label={getString('importRepo.url')}
                        placeholder={getString(getHostPlaceHolder(values.gitProvider))}
                        tooltipProps={{
                          dataTooltipId: 'spaceUserTextField'
                        }}
                        className={css.hostContainer}
                      />
                    )}
                    {formik.errors.host ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.host}
                      </Text>
                    ) : null}
                    <Layout.Horizontal flex>
                      {getString('importSpace.authorization')}
                      <Container padding={{ left: 'small' }} width={'100%'}>
                        <hr className={css.dividerContainer} />
                      </Container>
                    </Layout.Horizontal>
                    {[GitProviders.BITBUCKET, GitProviders.AZURE].includes(values.gitProvider) && (
                      <FormInput.Text
                        name="username"
                        label={getString('userName')}
                        placeholder={getString('importRepo.userPlaceholder')}
                        tooltipProps={{
                          dataTooltipId: 'spaceUserTextField'
                        }}
                      />
                    )}
                    {formik.errors.username ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.username}
                      </Text>
                    ) : null}
                    <FormInput.Text
                      name="password"
                      label={
                        [GitProviders.BITBUCKET, GitProviders.AZURE].includes(values.gitProvider)
                          ? getString('importRepo.appPassword')
                          : getString('importRepo.passToken')
                      }
                      placeholder={
                        [GitProviders.BITBUCKET, GitProviders.AZURE].includes(values.gitProvider)
                          ? getString('importRepo.appPasswordPlaceholder')
                          : getString('importRepo.passTokenPlaceholder')
                      }
                      tooltipProps={{
                        dataTooltipId: 'spacePasswordTextField'
                      }}
                      inputGroup={{ type: 'password' }}
                    />
                    {formik.errors.password ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.password}
                      </Text>
                    ) : null}
                  </Container>
                </>
              ) : null}
              {step === 1 ? (
                <>
                  <Layout.Horizontal flex>
                    <Text className={css.detailsLabel} font={{ size: 'small' }} flex>
                      {getString('importSpace.details')}
                    </Text>
                    <Container padding={{ left: 'small' }} width={'100%'}>
                      <hr className={css.dividerContainer} />
                    </Container>
                  </Layout.Horizontal>
                  <Container className={css.textContainer} width={'70%'}>
                    <FormInput.Text
                      name="organization"
                      label={getString(getOrgLabel(values.gitProvider))}
                      placeholder={getString(getOrgPlaceholder(values.gitProvider))}
                      tooltipProps={{
                        dataTooltipId: 'importSpaceOrgName'
                      }}
                      onChange={event => {
                        const target = event.target as HTMLInputElement
                        if (target.value) {
                          formik.validateField('organization')
                        }
                      }}
                    />
                    {formik.errors.organization ? (
                      <Text
                        margin={{ bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.organization}
                      </Text>
                    ) : null}
                    {values.gitProvider === GitProviders.AZURE && (
                      <FormInput.Text
                        name="project"
                        label={getString('importRepo.project')}
                        placeholder={getString('importRepo.projectPlaceholder')}
                        tooltipProps={{
                          dataTooltipId: 'importSpaceProjectName'
                        }}
                      />
                    )}
                    {formik.errors.project ? (
                      <Text
                        margin={{ top: 'small', bottom: 'small' }}
                        color={Color.RED_500}
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_500 }}>
                        {formik.errors.project}
                      </Text>
                    ) : null}
                    {standalone && (
                      <Layout.Horizontal>
                        <Label>{getString('importSpace.importLabel')}</Label>
                        <Icon padding={{ left: 'small' }} className={css.icon} name="code-info" size={16} />
                      </Layout.Horizontal>
                    )}
                    {standalone && (
                      <Container className={css.importContainer} padding={'medium'}>
                        <Layout.Horizontal>
                          <FormInput.CheckBox
                            name="repositories"
                            label={getString('pageTitle.repositories')}
                            tooltipProps={{
                              dataTooltipId: 'authorization'
                            }}
                            defaultChecked
                            onClick={() => {
                              setAuth(!auth)
                            }}
                            disabled
                            padding={{ right: 'small' }}
                            className={css.checkbox}
                          />
                          <Container padding={{ left: 'xxxlarge' }}>
                            <FormInput.CheckBox
                              name="importPipelineLabel"
                              label={getString('pageTitle.pipelines')}
                              tooltipProps={{
                                dataTooltipId: 'pipelines'
                              }}
                              onClick={() => {
                                setAuth(!auth)
                              }}
                            />
                          </Container>
                        </Layout.Horizontal>
                      </Container>
                    )}
                  </Container>
                </>
              ) : null}

              <hr className={css.dividerContainer} />

              <Layout.Horizontal
                spacing="small"
                padding={{ right: 'xxlarge', bottom: 'large' }}
                style={{ alignItems: 'center' }}>
                {step === 1 ? (
                  <Button
                    disabled={buttonLoading}
                    text={
                      buttonLoading ? (
                        <>
                          <Container className={css.loadingIcon} width={93.5} flex={{ alignItems: 'center' }}>
                            <Icon className={css.loadingIcon} name="steps-spinner" size={16} />
                          </Container>
                        </>
                      ) : (
                        getString('importRepos.title')
                      )
                    }
                    intent={Intent.PRIMARY}
                    onClick={() => {
                      handleValidationClick()
                      if (formik.values.name !== '' && formik.values.organization !== '') {
                        handleImport()
                        setButtonLoading(false)
                      }
                      formik.setErrors({})
                    }}
                  />
                ) : (
                  <Button
                    text={getString('importSpace.next')}
                    intent={Intent.PRIMARY}
                    onClick={() => {
                      handleValidationClick()
                      if (
                        (!formik.errors.gitProvider && formik.touched.gitProvider) ||
                        (!formik.errors.username && formik.touched.username) ||
                        (!formik.errors.password && formik.touched.password)
                      ) {
                        formik.setErrors({})
                        setStep(1)
                      }
                    }}
                  />
                )}

                <Button text={getString('cancel')} minimal onClick={hideModal} />
                <FlexExpander />

                {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
              </Layout.Horizontal>
            </FormikForm>
          </Container>
        )
      }}
    </Formik>
  )
}

export default ImportReposForm
