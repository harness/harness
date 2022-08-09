import React from 'react'
import { useHistory } from 'react-router-dom'
import {
  Button,
  ButtonVariation,
  Container,
  Layout,
  Page,
  useModalHook,
  Formik,
  FormikForm,
  FormInput,
  useToaster
} from '@harness/uicore'
import { Dialog } from '@blueprintjs/core'
import { useListPipelines, useCreatePipeline, useDeletePipeline } from 'services/pm'
import { useStrings } from 'framework/strings'
import Table from '../../components/Table/Table'
import routes from 'RouteDefinitions'

import styles from './Pipelines.module.scss'

interface PipelineForm {
  name: string
  desc: string
}

export const Home: React.FC = () => {
  const { getString } = useStrings()
  const history = useHistory()
  const { showError, showSuccess } = useToaster()
  const { mutate: createPipeline } = useCreatePipeline({})
  const { mutate: deletePipeline } = useDeletePipeline({})
  const { data: pipelineList, loading, error, refetch } = useListPipelines({})
  const modalProps = {
    isOpen: true,
    usePortal: true,
    autoFocus: true,
    canEscapeKeyClose: true,
    canOutsideClickClose: true,
    enforceFocus: true,
    title: 'Add New Pipeline',
    style: { width: 400, height: 300 }
  }

  const onRowClick = (pipeline: string) => {
    history.push(routes.toPipeline({ pipeline }))
  }

  const onSettingsClick = (pipeline: string) => {
    history.push(routes.toPipelineSettings({ pipeline }))
  }

  const [openModal, hideModal] = useModalHook(() => (
    <Dialog onClose={hideModal} {...modalProps}>
      <Container margin={{ top: 'large' }} flex={{ alignItems: 'center', justifyContent: 'space-around' }}>
        <Formik<PipelineForm> initialValues={{ name: '', desc: '' }} formName="newPipelineForm" onSubmit={handleSubmit}>
          <FormikForm>
            <FormInput.Text name="name" label={getString('common.name')} />
            <FormInput.Text name="desc" label={getString('common.description')} />
            <Button type="submit" intent="primary" width="100%">
              Create
            </Button>
          </FormikForm>
        </Formik>
      </Container>
    </Dialog>
  ))

  const handleCreate = async (data: PipelineForm) => {
    const { name, desc } = data
    try {
      await createPipeline({ name, desc })
      showSuccess(getString('common.itemCreated'))
      refetch()
    } catch (err) {
      showError(`Error: ${err}`)
      console.error({ err })
    }
  }

  const handleSubmit = (data: PipelineForm): void => {
    handleCreate(data)
    hideModal()
  }

  if (error) {
    history.push(routes.toLogin())
  }

  return (
    <Container className={styles.root} height="inherit">
      <Page.Header title={getString('pipelines')} />
      <Layout.Horizontal spacing="large" className={styles.header}>
        <Button variation={ButtonVariation.PRIMARY} text="New Pipeline" icon="plus" onClick={openModal} />
        <div style={{ flex: 1 }} />
      </Layout.Horizontal>
      <Page.Body
        loading={loading}
        retryOnError={() => refetch()}
        error={(error?.data as Error)?.message || error?.message}>
        <Table
          onRowClick={onRowClick}
          refetch={refetch}
          data={pipelineList}
          onDelete={deletePipeline}
          onSettingsClick={onSettingsClick}
        />
      </Page.Body>
    </Container>
  )
}
