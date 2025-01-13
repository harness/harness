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

import {
  Button,
  ButtonVariation,
  Container,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  Text,
  useToaster
} from '@harnessio/uicore'
import React from 'react'
import { useHistory } from 'react-router-dom'
import { Color, Intent } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { String, useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage } from 'utils/Utils'
import type { OpenapiUpdatePipelineRequest, TypesPipeline } from 'services/code'
import css from './PipelineSettingsTab.module.scss'

interface SettingsContentProps {
  pipeline: string
  repoPath: string
  yamlPath: string
}

interface SettingsFormData {
  name: string
  yamlPath: string
}

const PipelineSettingsTab = ({ pipeline, repoPath, yamlPath }: SettingsContentProps) => {
  const { getString } = useStrings()
  const { mutate: updatePipeline } = useMutate<TypesPipeline>({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}`
  })
  const { mutate: deletePipeline } = useMutate<TypesPipeline>({
    verb: 'DELETE',
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}`
  })
  const { showSuccess, showError, clear: clearToaster } = useToaster()
  const confirmDeletePipeline = useConfirmAct()
  const history = useHistory()

  return (
    <Layout.Vertical padding={'medium'} spacing={'medium'}>
      <Container padding={'large'} className={css.generalContainer}>
        <Formik<SettingsFormData>
          initialValues={{
            name: pipeline,
            yamlPath
          }}
          formName="pipelineSettings"
          enableReinitialize={true}
          validationSchema={yup.object().shape({
            name: yup
              .string()
              .trim()
              .required(`${getString('name')} ${getString('isRequired')}`),
            yamlPath: yup
              .string()
              .trim()
              .required(`${getString('pipelines.yamlPath')} ${getString('isRequired')}`)
          })}
          validateOnChange
          validateOnBlur
          onSubmit={async formData => {
            const { name, yamlPath: newYamlPath } = formData
            try {
              const payload: OpenapiUpdatePipelineRequest = {
                config_path: newYamlPath,
                identifier: name
              }
              await updatePipeline(payload, {
                pathParams: { path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}` }
              })
              history.push(
                routes.toCODEPipelineSettings({
                  repoPath,
                  pipeline: name
                })
              )
              clearToaster()
              showSuccess(getString('pipelines.updatePipelineSuccess', { pipeline }))
            } catch (exception) {
              clearToaster()
              showError(getErrorMessage(exception), 0, 'pipelines.failedToUpdatePipeline')
            }
          }}>
          {() => {
            return (
              <FormikForm>
                <Layout.Vertical spacing={'large'}>
                  <Layout.Horizontal spacing={'large'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                    <FormInput.Text
                      name="name"
                      className={css.textContainer}
                      label={
                        <Text color={Color.GREY_800} font={{ size: 'small' }}>
                          {getString('name')}
                        </Text>
                      }
                    />
                  </Layout.Horizontal>
                  <Layout.Horizontal spacing={'large'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                    <FormInput.Text
                      name="yamlPath"
                      className={css.textContainer}
                      label={
                        <Text color={Color.GREY_800} font={{ size: 'small' }}>
                          {getString('pipelines.yamlPath')}
                        </Text>
                      }
                    />
                  </Layout.Horizontal>
                  <div className={css.separator} />
                  <Layout.Horizontal spacing={'large'}>
                    <Button intent={Intent.PRIMARY} type="submit" text={getString('save')} />
                    <Button variation={ButtonVariation.TERTIARY} type="reset" text={getString('cancel')} />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </FormikForm>
            )
          }}
        </Formik>
      </Container>
      <Container padding={'large'} className={css.generalContainer}>
        <Layout.Vertical>
          <Text icon="main-trash" color={Color.GREY_600} font={{ size: 'normal' }}>
            {getString('dangerDeletePipeline')}
          </Text>
          <Layout.Horizontal padding={{ top: 'medium', left: 'medium' }} flex={{ justifyContent: 'space-between' }}>
            <Container intent="warning" padding={'small'} className={css.yellowContainer}>
              <Text
                icon="main-issue"
                iconProps={{ size: 18, color: Color.ORANGE_700, margin: { right: 'small' } }}
                color={Color.WARNING}>
                {getString('pipelines.deletePipelineWarning', {
                  pipeline
                })}
              </Text>
            </Container>
            <Button
              margin={{ right: 'medium' }}
              intent={Intent.DANGER}
              onClick={() => {
                confirmDeletePipeline({
                  title: getString('pipelines.deletePipelineButton'),
                  confirmText: getString('delete'),
                  intent: Intent.DANGER,
                  message: <String useRichText stringID="pipelines.deletePipelineConfirm" vars={{ pipeline }} />,
                  action: async () => {
                    try {
                      await deletePipeline(null)
                      history.push(
                        routes.toCODEPipelines({
                          repoPath
                        })
                      )
                      showSuccess(getString('pipelines.deletePipelineSuccess', { pipeline }))
                    } catch (e) {
                      showError(getString('pipelines.deletePipelineError'))
                    }
                  }
                })
              }}
              variation={ButtonVariation.PRIMARY}
              text={getString('pipelines.deletePipelineButton')}></Button>
          </Layout.Horizontal>
        </Layout.Vertical>
      </Container>
    </Layout.Vertical>
  )
}

export default PipelineSettingsTab
