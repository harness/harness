import React from 'react'
import { useHistory, Link, useLocation } from 'react-router-dom'
import {
  FormInput,
  Formik,
  FormikForm,
  Button,
  Text,
  Color,
  Container,
  HarnessIcons,
  Layout,
  useToaster
} from '@harness/uicore'
import { get } from 'lodash-es'
import { useAPIToken } from 'hooks/useAPIToken'
import { useOnLogin, useOnRegister } from 'services/pm'
import { useStrings } from 'framework/strings'
import routes from 'RouteDefinitions'

import styles from './Login.module.scss'

interface LoginForm {
  username: string
  password: string
}

const HarnessLogo = HarnessIcons['harness-logo-black']

export const Login: React.FC = () => {
  const history = useHistory()
  const { getString } = useStrings()
  const { pathname } = useLocation()
  const [, setToken] = useAPIToken()
  const { mutate } = useOnLogin({})
  const { showError } = useToaster()
  const { mutate: mutateRegister } = useOnRegister({})

  const handleLogin = async (data: LoginForm): Promise<void> => {
    const formData = new FormData()
    formData.append('username', data.username)
    formData.append('password', data.password)

    if (pathname === '/login') {
      mutate(formData as unknown as void)
        .then(_data => {
          setToken(get(_data, 'access_token' as string))
          history.replace(routes.toPolicyDashboard())
        })
        .catch(error => {
          showError(`Error: ${error}`)
        })
    } else {
      mutateRegister(formData as unknown as void)
        .then(_data => {
          setToken(get(_data, 'access_token' as string))
          history.replace(routes.toPolicyDashboard())
        })
        .catch(error => {
          showError(`Error: ${error}`)
        })
    }
  }

  const handleSubmit = (data: LoginForm): void => {
    handleLogin(data)
  }

  return (
    <div className={styles.layout}>
      <div className={styles.cardColumn}>
        <div className={styles.card}>
          <Container flex={{ justifyContent: 'space-between', alignItems: 'center' }} margin={{ bottom: 'xxxlarge' }}>
            <HarnessLogo height={25} />
          </Container>
          <Text font={{ size: 'large', weight: 'bold' }} color={Color.BLACK}>
            {pathname === '/login' ? getString('signIn') : getString('signUp')}
          </Text>
          <Text font={{ size: 'medium' }} color={Color.BLACK} margin={{ top: 'xsmall' }}>
            and get ship done.
          </Text>

          <Container margin={{ top: 'xxxlarge' }}>
            <Formik<LoginForm>
              initialValues={{ username: '', password: '' }}
              formName="loginPageForm"
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text name="email" label={getString('email')} />
                <FormInput.Text name="password" label={getString('password')} inputGroup={{ type: 'password' }} />
                <Button type="submit" intent="primary" width="100%">
                  {pathname === '/login' ? getString('signIn') : getString('signUp')}
                </Button>
              </FormikForm>
            </Formik>
          </Container>

          <Layout.Horizontal margin={{ top: 'xxxlarge' }} spacing="xsmall">
            <Text>{pathname === '/login' ? getString('noAccount') : getString('existingAccount')}</Text>
            <Link to={pathname === '/login' ? routes.toRegister() : routes.toSignIn()}>
              {pathname === '/login' ? getString('signUp') : getString('signIn')}
            </Link>
          </Layout.Horizontal>
        </div>
      </div>
      <div className={styles.imageColumn}>
        <img
          className={styles.image}
          src={'https://app.harness.io/auth/assets/AuthIllustration.f611adb8.svg'}
          alt=""
          aria-hidden
        />
      </div>
    </div>
  )
}
