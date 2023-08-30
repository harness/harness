import React from 'react'
import { useHistory } from 'react-router-dom'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { Button, ButtonVariation, Dialog, FormInput, Formik, FormikForm, Layout, useToaster } from '@harnessio/uicore'
import { useModalHook } from 'hooks/useModalHook'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { OpenapiCreatePipelineRequest, TypesPipeline, TypesRepository } from 'services/code'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'

interface FormData {
  name: string
  branch: string
  yamlPath: string
}

const useNewPipelineModal = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const history = useHistory()
  const { showError } = useToaster()

  const repoMetadata: TypesRepository = { path: `${space}/vb-repo`, default_branch: 'main' }

  const { mutate: savePipeline } = useMutate<TypesPipeline>({
    verb: 'POST',
    path: `/api/v1/pipelines`
  })

  const handleCreatePipeline = (formData: FormData): void => {
    const { name, branch, yamlPath } = formData
    try {
      const payload: OpenapiCreatePipelineRequest = {
        config_path: yamlPath,
        default_branch: branch,
        space_ref: space,
        repo_ref: repoMetadata.path,
        repo_type: 'GITNESS',
        uid: name
      }
      savePipeline(payload)
        .then(() => {
          history.push(routes.toCODEPipelineEdit({ space, pipeline: name }))
          hideModal()
        })
        .catch(_error => {
          showError(getErrorMessage(_error), 0, 'pipelines.failedToCreatePipeline')
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
          initialValues={{ name: '', branch: repoMetadata.default_branch || '', yamlPath: '' }}
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
          <FormikForm>
            <Layout.Vertical spacing="small">
              <Layout.Vertical spacing="small">
                <FormInput.Text
                  name="name"
                  label={getString('name')}
                  placeholder={getString('pipelines.enterPipelineName')}
                  inputGroup={{ autoFocus: true }}
                />
                <FormInput.Text name="branch" label={getString('pipelines.basedOn')} />
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
        </Formik>
      </Dialog>
    )
  }, [])

  return {
    openModal,
    hideModal
  }
}

export default useNewPipelineModal
