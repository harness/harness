import React, { useMemo, useState } from 'react'
import { useHistory } from 'react-router-dom'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
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
import { useModalHook } from 'hooks/useModalHook'
import type { OpenapiCreatePipelineRequest, TypesPipeline, TypesRepository } from 'services/code'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { DEFAULT_YAML_PATH_PREFIX, DEFAULT_YAML_PATH_SUFFIX } from '../../pages/AddUpdatePipeline/Constants'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { FontVariation } from '@harnessio/design-system'
import { capitalize } from 'lodash'

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
  const [repo, setRepo] = useState<TypesRepository | undefined>()
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
        uid: name
      }
      savePipeline(payload, { pathParams: { path: `/api/v1/repos/${repoPath}/+/pipelines` } })
        .then(() => {
          history.push(routes.toCODEPipelineEdit({ repoPath, pipeline: name }))
          hideModal()
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
    openModal: ({ repoMetadata }: { repoMetadata?: TypesRepository }) => {
      setRepo(repoMetadata)
      openModal()
    },
    hideModal
  }
}

export default useNewPipelineModal
