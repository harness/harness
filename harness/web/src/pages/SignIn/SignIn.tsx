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

import React, { useCallback } from 'react'
import {
  Button,
  Container,
  FlexExpander,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  Text,
  useToaster,
  StringSubstitute
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import * as Yup from 'yup'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useOnLogin } from 'services/code'
import AuthLayout from 'components/AuthLayout/AuthLayout'
import { useAppContext } from 'AppContext'
import { getErrorMessage, type LoginForm } from 'utils/Utils'
import css from './SignIn.module.scss'

export const SignIn: React.FC = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { mutate } = useOnLogin({
    queryParams: {
      include_cookie: true
    }
  })
  const { showError } = useToaster()
  const onLogin = useCallback(
    ({ username, password }: LoginForm) => {
      mutate(
        { login_identifier: username, password },
        {
          headers: { Authorization: '' }
        }
      )
        .then(() => {
          window.location.replace(window.location.origin + routes.toCODEHome())
        })
        .catch(error => {
          showError(getErrorMessage(error))
        })
    },
    [mutate, showError, routes]
  )
  const onSubmit = useCallback(
    (data: LoginForm): void => {
      if (data.username && data.password) {
        onLogin(data)
      }
    },
    [onLogin]
  )

  return (
    <AuthLayout>
      <Container className={css.signInContainer}>
        <Layout.Horizontal flex={{ alignItems: 'center' }}>
          <Text font={{ size: 'large', weight: 'bold' }} color={Color.BLACK}>
            {getString('signIn')}
          </Text>
          <FlexExpander />
          <Layout.Horizontal spacing="xsmall">
            <Text>{getString('noAccount?')}</Text>
            <Link to={routes.toRegister()}>{getString('signUp')}</Link>
          </Layout.Horizontal>
        </Layout.Horizontal>

        <Container margin={{ top: 'xxlarge' }}>
          <Formik<LoginForm>
            initialValues={{ username: '', password: '' }}
            formName="loginPageForm"
            onSubmit={onSubmit}
            validationSchema={Yup.object().shape({
              username: Yup.string().required(getString('userNameRequired')),
              password: Yup.string().required(getString('passwordRequired'))
            })}>
            <FormikForm>
              <FormInput.Text name="username" label={getString('emailUser')} disabled={false} />
              <FormInput.Text
                name="password"
                label={getString('password')}
                inputGroup={{ type: 'password' }}
                disabled={false}
              />
              <Button type="submit" intent="primary" loading={false} disabled={false} width="100%">
                {getString('signIn')}
              </Button>
            </FormikForm>
          </Formik>
        </Container>
        <Layout.Horizontal padding={{ top: 'medium' }} spacing="xsmall">
          <Text>
            <StringSubstitute
              str={getString('bySigningIn')}
              vars={{
                policy: <a href="https://harness.io/privacy"> {getString('privacyPolicy')} </a>,
                terms: <a href="https://harness.io/subscriptionterms"> {getString('termsOfUse')} </a>
              }}
            />
          </Text>
        </Layout.Horizontal>
      </Container>
    </AuthLayout>
  )
}
