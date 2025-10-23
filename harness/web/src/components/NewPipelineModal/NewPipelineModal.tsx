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

import React, { useMemo, useState } from 'react'
import { useHistory } from 'react-router-dom'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { capitalize } from 'lodash-es'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  Text,
  useToaster
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useModalHook } from 'hooks/useModalHook'
import type { OpenapiCreatePipelineRequest, TypesPipeline, RepoRepositoryOutput } from 'services/code'
import { useStrings } from 'framework/strings'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { DEFAULT_YAML_PATH_PREFIX, DEFAULT_YAML_PATH_SUFFIX } from '../../pages/AddUpdatePipeline/Constants'

import css from './NewPipelineModal.module.scss'

interface FormData {
  name: string
  branch: string
  yamlPath: string
}

const useNewPipelineModal = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const history = useHistory()
  const { showError } = useToaster()
  const [repo, setRepo] = useState<RepoRepositoryOutput | undefined>()
  const repoPath = useMemo(() => repo?.path || '', [repo])

  const { mutate: savePipeline } = useMutate<TypesPipeline>({
    verb: 'POST',
    path: `/api/v1/repos/${repoPath}/+/pipelines`
  })

  const handleCreatePipeline = (formData: FormData): void => {
    const { name, branch, yamlPath } = formData
    try {
      const payload: OpenapiCreatePipelineRequest = {
        config_path: yamlPath,
        default_branch: branch,
        identifier: name
      }
      savePipeline(payload, { pathParams: { path: `/api/v1/repos/${repoPath}/+/pipelines` } })
        .then(() => {
          hideModal()
          history.push(routes.toCODEPipelineEdit({ repoPath, pipeline: name }))
        })
        .catch(error => {
          showError(getErrorMessage(error), 0, 'pipelines.failedToCreatePipeline')
        })
    } catch (exception) {
      showError(getErrorMessage(exception), 0, 'pipelines.failedToCreatePipeline')
    }
  }

  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      hideModal()
    }
    return (
      <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('pipelines.createNewPipeline')}>
        <Formik<FormData>
          initialValues={{ name: '', branch: repo?.default_branch || '', yamlPath: '' }}
          formName="createNewPipeline"
          enableReinitialize={true}
          validationSchema={yup.object().shape({
            name: yup
              .string()
              .trim()
              .required(`${getString('name')} ${getString('isRequired')}`),
            branch: yup
              .string()
              .trim()
              .required(`${getString('branch')} ${getString('isRequired')}`),
            yamlPath: yup
              .string()
              .trim()
              .required(`${getString('pipelines.yamlPath')} ${getString('isRequired')}`)
          })}
          validateOnChange
          validateOnBlur
          onSubmit={handleCreatePipeline}>
          {formik => {
            return (
              <FormikForm>
                <Layout.Vertical spacing="small">
                  <Layout.Vertical spacing="small">
                    <FormInput.Text
                      name="name"
                      label={getString('name')}
                      placeholder={getString('pipelines.enterPipelineName')}
                      inputGroup={{ autoFocus: true }}
                      onChange={event => {
                        const input = (event.target as HTMLInputElement)?.value
                        formik?.setFieldValue('name', input)
                        if (input) {
                          // Keeping minimal validation for now, this could be much more exhaustive
                          const path = input.trim().replace(/\s/g, '')
                          formik?.setFieldValue(
                            'yamlPath',
                            DEFAULT_YAML_PATH_PREFIX.concat(path).concat(DEFAULT_YAML_PATH_SUFFIX)
                          )
                        }
                      }}
                    />
                    <Layout.Vertical spacing="xsmall" padding={{ bottom: 'medium' }}>
                      <Text font={{ variation: FontVariation.BODY }}>{capitalize(getString('branch'))}</Text>
                      <Container className={css.branchSelect}>
                        <BranchTagSelect
                          gitRef={formik?.values?.branch || repo?.default_branch || ''}
                          onSelect={(ref: string) => {
                            formik?.setFieldValue('branch', ref)
                          }}
                          repoMetadata={repo || {}}
                          disableBranchCreation
                          disableViewAllBranches
                          forBranchesOnly
                        />
                      </Container>
                    </Layout.Vertical>
                    <FormInput.Text
                      name="yamlPath"
                      label={getString('pipelines.yamlPath')}
                      placeholder={getString('pipelines.enterYAMLPath')}
                    />
                  </Layout.Vertical>
                  <Layout.Horizontal spacing="medium" width="100%">
                    <Button variation={ButtonVariation.PRIMARY} text={getString('create')} type="submit" />
                    <Button variation={ButtonVariation.SECONDARY} text={getString('cancel')} onClick={onClose} />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </FormikForm>
            )
          }}
        </Formik>
      </Dialog>
    )
  }, [repo])

  return {
    openModal: ({ repoMetadata }: { repoMetadata?: RepoRepositoryOutput }) => {
      setRepo(repoMetadata)
      openModal()
    },
    hideModal
  }
}

export default useNewPipelineModal
