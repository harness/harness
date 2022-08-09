import React, { useState } from 'react'
import {
  Container,
  Button,
  Formik,
  FormikForm,
  FormInput,
  Text,
  Color,
  Layout,
  ButtonVariation,
  Page,
  CodeBlock
} from '@harness/uicore'
import { useAPIToken } from 'hooks/useAPIToken'
import { useStrings } from 'framework/strings'

import styles from './Settings.module.scss'

interface FormValues {
  name?: string
  desc?: string
}

interface FormProps {
  name?: string
  desc?: string
  handleSubmit: (values: FormValues) => void
  loading: boolean | undefined
  refetch: () => void
  handleDelete: () => void
  error?: any
  title: string
}

export const Settings = ({ name, desc, handleSubmit, handleDelete, loading, refetch, error, title }: FormProps) => {
  const [token] = useAPIToken()
  const { getString } = useStrings()
  const [showToken, setShowToken] = useState(false)
  const [editDetails, setEditDetails] = useState(false)

  const onSubmit = (values: FormValues) => {
    handleSubmit(values)
    setEditDetails(false)
  }

  const editForm = (
    <Formik initialValues={{ name, desc }} formName="newPipelineForm" onSubmit={values => onSubmit(values)}>
      <FormikForm>
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
          <Text color={Color.GREY_600} className={styles.minWidth}>
            {getString('common.name')}
          </Text>
          <FormInput.Text name="name" className={styles.input} />
        </Layout.Horizontal>
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
          <Text color={Color.GREY_600} className={styles.minWidth}>
            {getString('common.description')}
          </Text>
          <FormInput.Text name="desc" className={styles.input} />
        </Layout.Horizontal>
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
          <Button variation={ButtonVariation.LINK} icon="updated" text={getString('common.save')} type="submit" />
          <Button variation={ButtonVariation.LINK} onClick={handleDelete}>
            Delete
          </Button>
        </Layout.Horizontal>
      </FormikForm>
    </Formik>
  )

  return (
    <Container className={styles.root} height="inherit">
      <Page.Header title={getString('settings')} />
      <Page.Body
        loading={loading}
        retryOnError={() => refetch()}
        error={(error?.data as Error)?.message || error?.message}>
        <Container margin="xlarge" padding="xlarge" className={styles.container} background="white">
          <Text color={Color.BLACK} font={{ weight: 'semi-bold', size: 'medium' }} margin={{ bottom: 'xlarge' }}>
            {title}
          </Text>
          {editDetails ? (
            editForm
          ) : (
            <>
              <Layout.Horizontal
                flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
                margin={{ bottom: 'large' }}>
                <Text color={Color.GREY_600} className={styles.minWidth}>
                  {getString('common.name')}
                </Text>
                <Text color={Color.GREY_800}>{name}</Text>
              </Layout.Horizontal>
              <Layout.Horizontal
                flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
                margin={{ bottom: 'large' }}>
                <Text className={styles.minWidth}>{getString('common.description')}</Text>
                <Text color={Color.GREY_800}>{desc}</Text>
              </Layout.Horizontal>
            </>
          )}
          {!editDetails && (
            <Button
              variation={ButtonVariation.LINK}
              icon="Edit"
              text={getString('common.edit')}
              onClick={() => setEditDetails(true)}
            />
          )}
        </Container>
        <Container margin="xlarge" padding="xlarge" className={styles.container} background="white">
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
            <Text className={styles.minWidth}>{getString('common.token')}</Text>
            <Button variation={ButtonVariation.LINK} onClick={() => setShowToken(!showToken)}>
              Display/Hide Token
            </Button>
          </Layout.Horizontal>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
            {showToken && <CodeBlock allowCopy format="pre" snippet={token} />}
          </Layout.Horizontal>
        </Container>
      </Page.Body>
    </Container>
  )
}
