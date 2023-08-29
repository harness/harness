import React, { useCallback } from 'react'
import { useHistory } from 'react-router-dom'
import { useMutate } from 'restful-react'
import { Button, ButtonVariation, Dialog, Layout } from '@harnessio/uicore'
import { useModalHook } from 'hooks/useModalHook'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { OpenapiCreatePipelineRequest, TypesPipeline } from 'services/code'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'

const useNewPipelineModal = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const history = useHistory()

  const { mutate: savePipeline } = useMutate<TypesPipeline>({
    verb: 'POST',
    path: `/api/v1/pipelines`
  })

  const handleSavePipeline = useCallback(async (): Promise<void> => {
    const randomToken = (Math.random() + 1).toString(36).substring(7)

    const payload: OpenapiCreatePipelineRequest = {
      config_path: `config_path_${randomToken}`,
      default_branch: 'main',
      space_ref: 'test-space',
      repo_ref: 'test-space/vb-repo',
      repo_type: 'GITNESS',
      uid: `pipeline_uid_${randomToken}`
    }
    await savePipeline(payload)
      .then(() => {
        history.push(routes.toCODEPipelinesNew({ space }))
        hideModal()
      })
      .catch(() => hideModal())
  }, [])

  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      hideModal()
    }
    return (
      <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('pipelines.createNewPipeline')}>
        <Layout.Vertical flex={{ justifyContent: 'center' }}>
          <Layout.Horizontal spacing="medium" flex={{ justifyContent: 'flex-start' }} width="100%">
            <Button variation={ButtonVariation.PRIMARY} text={getString('create')} onClick={handleSavePipeline} />
            <Button variation={ButtonVariation.SECONDARY} text={getString('cancel')} onClick={onClose} />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Dialog>
    )
  }, [])

  return {
    openModal,
    hideModal
  }
}

export default useNewPipelineModal
