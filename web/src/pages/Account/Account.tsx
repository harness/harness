import React, { useState, useEffect } from 'react'
import { useHistory } from 'react-router-dom'
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
  useToaster
} from '@harness/uicore'
import { useUpdateUser, useGetUser } from 'services/pm'
import { useStrings } from 'framework/strings'
import routes from 'RouteDefinitions'

import styles from './Account.module.scss'

interface UserFormProps {
  name: string
  email: string
  password1: string
  password2: string
}

export const Account = () => {
  const history = useHistory()
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const { data, loading, error, refetch } = useGetUser({})
  const { mutate } = useUpdateUser({})
  const [editDetails, setEditDetails] = useState(false)
  const [name, setName] = useState<string | undefined>('')
  const [email, setEmail] = useState<string | undefined>('')

  useEffect(() => {
    setName(data?.name)
    setEmail(data?.email)
  }, [data])

  if (error) {
    history.push(routes.toLogin())
  }

  const updateUserDetails = async ({ email, name, password1 }: UserFormProps) => {
    try {
      await mutate({ email, name, password: password1 })
      showSuccess(getString('common.itemUpdated'))
      refetch()
    } catch (err) {
      showError(`Error: ${err}`)
      console.error({ err })
    }
  }

  const handleSubmit = (data: UserFormProps): void => {
    setEditDetails(false)
    updateUserDetails(data)
  }

  const editUserForm = (
    <Formik<UserFormProps>
      initialValues={{ name: name as string, email: email as string, password1: '', password2: '' }}
      formName="newPipelineForm"
      onSubmit={handleSubmit}>
      <FormikForm>
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
          <Text color={Color.GREY_600} className={styles.minWidth}>
            {getString('common.name')}
          </Text>
          <FormInput.Text name="name" className={styles.input} />
        </Layout.Horizontal>
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
          <Text color={Color.GREY_600} className={styles.minWidth}>
            {getString('common.email')}
          </Text>
          <FormInput.Text name="email" className={styles.input} />
        </Layout.Horizontal>
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} margin={{ bottom: 'large' }}>
          <Text color={Color.GREY_600} className={styles.minWidth}>
            {getString('password')}
          </Text>
          <FormInput.Text
            name="password1"
            label="Password"
            inputGroup={{ type: 'password' }}
            className={styles.input}
          />
          <FormInput.Text
            name="password2"
            label="Re-type your Password"
            inputGroup={{ type: 'password' }}
            className={styles.input}
          />
        </Layout.Horizontal>
        <Button variation={ButtonVariation.LINK} icon="updated" text={getString('common.save')} type="submit" />
      </FormikForm>
    </Formik>
  )

  return (
    <Container className={styles.root} height="inherit">
      <Page.Header title={getString('common.accountOverview')} />
      <Page.Body
        loading={loading}
        retryOnError={() => refetch()}
        error={(error?.data as Error)?.message || error?.message}>
        <Container margin="xlarge" padding="xlarge" className={styles.container} background="white">
          <Text color={Color.BLACK} font={{ weight: 'semi-bold', size: 'medium' }} margin={{ bottom: 'xlarge' }}>
            {getString('common.accountDetails')}
          </Text>
          {editDetails ? (
            editUserForm
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
                <Text className={styles.minWidth}>{getString('common.email')}</Text>
                <Text color={Color.GREY_800}>{email}</Text>
              </Layout.Horizontal>
              <Layout.Horizontal
                flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
                margin={{ bottom: 'large' }}>
                <Text className={styles.minWidth}>{getString('password')}</Text>
                <Text padding={{ right: 'small' }} color={Color.GREY_800}>
                  *********
                </Text>
              </Layout.Horizontal>
              <Button
                variation={ButtonVariation.LINK}
                icon="Edit"
                text={getString('common.edit')}
                onClick={() => setEditDetails(true)}
              />
            </>
          )}
        </Container>
      </Page.Body>
    </Container>
  )
}
