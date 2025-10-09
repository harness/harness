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
import { useHistory } from 'react-router'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { capitalize } from 'lodash-es'
import { FontVariation } from '@harnessio/design-system'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  Formik,
  FormikForm,
  Layout,
  Text,
  useToaster
} from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { useModalHook } from 'hooks/useModalHook'
import type { CreateExecutionQueryParams, TypesExecution, RepoRepositoryOutput } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'

import css from './RunPipelineModal.module.scss'

interface FormData {
  branch: string
}

const useRunPipelineModal = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { showSuccess, showError, clear: clearToaster } = useToaster()
  const history = useHistory()
  const [repo, setRepo] = useState<RepoRepositoryOutput>()
  const [pipeline, setPipeline] = useState<string>('')
  const repoPath = useMemo(() => repo?.path || '', [repo])

  const { mutate: startExecution } = useMutate<TypesExecution>({
    verb: 'POST',
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/executions`
  })

  const runPipeline = async (formData: FormData): Promise<void> => {
    const { branch } = formData
    try {
      const response = await startExecution(
        {},
        {
          pathParams: { path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/executions` },
          queryParams: { branch } as CreateExecutionQueryParams
        }
      )
      clearToaster()
      showSuccess(getString('pipelines.executionStarted'))
      if (response?.number && !isNaN(response.number)) {
        history.push(routes.toCODEExecution({ repoPath, pipeline, execution: response.number.toString() }))
      }
      hideModal()
    } catch (error) {
      const errorMssg = getErrorMessage(error)
      const pipelineDoesNotExistOnGit = errorMssg === getString('pipelines.failedToFindPath')
      pipelineDoesNotExistOnGit
        ? showError(`${getString('pipelines.executionCouldNotStart')}, ${errorMssg}.`)
        : showError(getErrorMessage(error), 0, 'pipelines.executionCouldNotStart')
    }
  }

  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      hideModal()
    }
    return (
      <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('pipelines.run')}>
        <Formik
          formName="run-pipeline-form"
          initialValues={{ branch: repo?.default_branch || '' }}
          validationSchema={yup.object().shape({
            branch: yup
              .string()
              .trim()
              .required(`${getString('branch')} ${getString('isRequired')}`)
          })}
          onSubmit={runPipeline}
          enableReinitialize>
          {formik => {
            return (
              <FormikForm>
                <Layout.Vertical spacing="medium">
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
                  <Layout.Horizontal spacing="medium">
                    <Button variation={ButtonVariation.PRIMARY} type="submit" text={getString('pipelines.run')} />
                    <Button variation={ButtonVariation.SECONDARY} text={getString('cancel')} onClick={onClose} />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </FormikForm>
            )
          }}
        </Formik>
      </Dialog>
    )
  }, [repo?.default_branch, pipeline])

  return {
    openModal: ({ repoMetadata, pipeline: pipelineUid }: { repoMetadata: RepoRepositoryOutput; pipeline: string }) => {
      setRepo(repoMetadata)
      setPipeline(pipelineUid)
      openModal()
    },
    hideModal
  }
}

export default useRunPipelineModal
