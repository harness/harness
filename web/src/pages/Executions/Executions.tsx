import React from 'react'
import { useParams, useHistory } from 'react-router-dom'
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
import { useListExecutions, useCreateExecution, useDeleteExecution } from 'services/pm'
import { startCase, camelCase } from 'lodash'
import { useStrings } from 'framework/strings'
import Table from '../../components/Table/Table'
import routes from 'RouteDefinitions'
import styles from './Executions.module.scss'

export interface ExecutionsParams {
  pipeline: string
}

interface ExecutionForm {
  name: string
  desc: string
}

export const Executions: React.FC = () => {
  const history = useHistory()
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const { pipeline } = useParams<ExecutionsParams>()
  const { mutate: deleteExecution } = useDeleteExecution({ pipeline: pipeline })
  const { mutate: createExecution } = useCreateExecution({ pipeline })
  const { data: executionList, loading, error, refetch } = useListExecutions({ pipeline })

  const title = `${startCase(camelCase(pipeline.replace(/-/g, ' ')))} ${getString('executions')}`

  const handleCreate = async ({ name, desc }: ExecutionForm) => {
    try {
      await createExecution({ name, desc })
      showSuccess(getString('common.itemCreated'))
      refetch()
    } catch (err) {
      showError(`Error: ${err}`)
      console.error({ error })
    }
  }

  const modalProps = {
    isOpen: true,
    usePortal: true,
    autoFocus: true,
    canEscapeKeyClose: true,
    canOutsideClickClose: true,
    enforceFocus: true,
    title: getString('addExecution'),
    style: { width: 400, height: 300 }
  }

  const handleSubmit = (data: ExecutionForm): void => {
    handleCreate(data)
    hideModal()
  }

  const onRowClick = (execution: string) => {
    history.push(routes.toPipelineExecutionSettings({ pipeline, execution }))
  }

  const onSettingsClick = (execution: string) => {
    history.push(routes.toPipelineExecutionSettings({ pipeline, execution }))
  }

  const [openModal, hideModal] = useModalHook(() => (
    <Dialog onClose={hideModal} {...modalProps}>
      <Container margin={{ top: 'large' }} flex={{ alignItems: 'center', justifyContent: 'space-around' }}>
        <Formik<ExecutionForm>
          initialValues={{ name: '', desc: '' }}
          formName="newExecutionForm"
          onSubmit={handleSubmit}>
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

  return (
    <Container className={styles.root} height="inherit">
      <Page.Header title={title} />
      <Layout.Horizontal spacing="large" className={styles.header}>
        <Button variation={ButtonVariation.PRIMARY} text="New Execution" icon="plus" onClick={openModal} />
        <div style={{ flex: 1 }} />
      </Layout.Horizontal>
      <Page.Body
        loading={loading}
        retryOnError={() => refetch()}
        error={(error?.data as Error)?.message || error?.message}>
        <Table
          onRowClick={onRowClick}
          refetch={refetch}
          data={executionList}
          onDelete={deleteExecution}
          onSettingsClick={onSettingsClick}
        />
      </Page.Body>
    </Container>
  )
}
