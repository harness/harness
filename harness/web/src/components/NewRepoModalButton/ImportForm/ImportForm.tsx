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
import { Button, Layout, FlexExpander, Formik, FormikForm, FormInput, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { REGEX_VALID_REPO_NAME } from 'utils/Utils'
import { ImportFormData, GitProviders, getProviders, getOrgLabel, getOrgPlaceholder } from 'utils/GitUtils'
import css from '../NewRepoModalButton.module.scss'

interface ImportFormProps {
  handleSubmit: (data: ImportFormData) => void
  loading: boolean // eslint-disable-next-line @typescript-eslint/no-explicit-any
  hideModal: any
}

const ImportForm = (props: ImportFormProps) => {
  const { standalone } = useAppContext()
  const { handleSubmit, loading, hideModal } = props
  const { getString } = useStrings()
  const [auth, setAuth] = useState(false)

  // eslint-disable-next-line no-control-regex
  // const MATCH_REPOURL_REGEX = /^(https?:\/\/(?:www\.)?(github|gitlab)\.com\/([^/]+\/[^/]+))/

  const formInitialValues: ImportFormData = {
    gitProvider: GitProviders.GITHUB,
    hostUrl: '',
    org: '',
    project: '',
    repo: '',
    username: '',
    password: '',
    name: '',
    description: '',
    importPipelineLabel: false
  }

  const validationSchemaStepOne = yup.object().shape({
    gitProvider: yup.string().required(),
    hostUrl: yup
      .string()
      // .matches(MATCH_REPOURL_REGEX, getString('importSpace.invalidUrl'))
      .when('gitProvider', {
        is: gitProvider =>
          ![GitProviders.GITHUB, GitProviders.GITLAB, GitProviders.BITBUCKET, GitProviders.AZURE].includes(gitProvider),
        then: yup.string().required(getString('importRepo.required')),
        otherwise: yup.string().notRequired() // Optional based on your needs
      }),
    org: yup
      .string()
      .trim()
      .when('gitProvider', {
        is: GitProviders.AZURE,
        then: yup.string().required(getString('importSpace.orgRequired'))
      }),
    project: yup
      .string()
      .trim()
      .when('gitProvider', {
        is: GitProviders.AZURE,
        then: yup.string().required(getString('importSpace.spaceNameRequired'))
      }),
    name: yup
      .string()
      .trim()
      .required(getString('validation.nameIsRequired'))
      .matches(REGEX_VALID_REPO_NAME, getString('validation.repoNamePatternIsNotValid'))
  })

  return (
    <Formik onSubmit={handleSubmit} initialValues={formInitialValues} formName="importRepoForm">
      {formik => {
        const { values } = formik
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
            <FormInput.Select
              name={'gitProvider'}
              label={getString('importSpace.gitProvider')}
              items={getProviders()}
              // className={css.selectBox}
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
                className={css.hideContainer}
                name="hostUrl"
                label={getString('importRepo.url')}
                placeholder={getString('importRepo.urlPlaceholder')}
                tooltipProps={{
                  dataTooltipId: 'repositoryURLTextField'
                }}
              />
            )}
            {formik.errors.hostUrl ? (
              <Text
                margin={{ top: 'small', bottom: 'small' }}
                color={Color.RED_500}
                icon="circle-cross"
                iconProps={{ color: Color.RED_500 }}>
                {formik.errors.hostUrl}
              </Text>
            ) : null}
            <FormInput.Text
              className={css.hideContainer}
              name="org"
              label={getString(getOrgLabel(values.gitProvider))}
              placeholder={getString(getOrgPlaceholder(values.gitProvider))}
            />
            {formik.errors.org ? (
              <Text
                margin={{ top: 'small', bottom: 'small' }}
                color={Color.RED_500}
                icon="circle-cross"
                iconProps={{ color: Color.RED_500 }}>
                {formik.errors.org}
              </Text>
            ) : null}
            {values.gitProvider === GitProviders.AZURE && (
              <FormInput.Text
                className={css.hideContainer}
                name="project"
                label={getString('importRepo.project')}
                placeholder={getString('importRepo.projectPlaceholder')}
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
            <FormInput.Text
              className={css.hideContainer}
              name="repo"
              label={getString('importRepo.repo')}
              placeholder={getString('importRepo.repoPlaceholder')}
              onChange={event => {
                const target = event.target as HTMLInputElement
                formik.setFieldValue('repo', target.value)
                if (target.value) {
                  formik.setFieldValue('name', target.value)
                  formik.validateField('repo')
                }
              }}
            />
            {formik.errors.repo ? (
              <Text
                margin={{ top: 'small', bottom: 'small' }}
                color={Color.RED_500}
                icon="circle-cross"
                iconProps={{ color: Color.RED_500 }}>
                {formik.errors.repo}
              </Text>
            ) : null}
            <Layout.Horizontal spacing="medium">
              <FormInput.CheckBox
                name="authorization"
                label={getString('importRepo.reqAuth')}
                tooltipProps={{
                  dataTooltipId: 'authorization'
                }}
                onClick={() => {
                  setAuth(!auth)
                }}
                style={auth ? {} : { margin: 0 }}
              />
              {standalone && (
                <FormInput.CheckBox
                  name="importPipelineLabel"
                  label={getString('pipelines.import')}
                  tooltipProps={{
                    dataTooltipId: 'pipelines'
                  }}
                  style={auth ? {} : { margin: 0 }}
                />
              )}
            </Layout.Horizontal>

            {auth ? (
              <>
                {[GitProviders.BITBUCKET, GitProviders.AZURE].includes(values.gitProvider) && (
                  <FormInput.Text
                    name="username"
                    label={getString('userName')}
                    placeholder={getString('importRepo.userPlaceholder')}
                    tooltipProps={{
                      dataTooltipId: 'repositoryUserTextField'
                    }}
                  />
                )}
                <FormInput.Text
                  inputGroup={{ type: 'password' }}
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
