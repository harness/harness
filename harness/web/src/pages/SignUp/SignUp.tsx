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
  StringSubstitute,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import * as Yup from 'yup'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import AuthLayout from 'components/AuthLayout/AuthLayout'
import { useAppContext } from 'AppContext'
import { getErrorMessage, type RegisterForm } from 'utils/Utils'
import { useOnRegister } from 'services/code'
import css from './SignUp.module.scss'

export const SignUp: React.FC = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()

  const { mutate } = useOnRegister({
    queryParams: {
      include_cookie: true
    }
  })
  const onRegister = useCallback(
    (data: RegisterForm) => {
      mutate(
        {
          display_name: data.username,
          email: data.email,
          uid: data.username,
          password: data.password
        },
        {
          headers: { Authorization: '' }
        }
      )
        .then(() => {
          showSuccess(getString('userCreated'))
          window.location.replace(window.location.origin + routes.toCODEHome())
        })
        .catch(error => {
          showError(getErrorMessage(error))
        })
    },
    [mutate, showSuccess, showError, getString, routes]
  )

  const handleSubmit = (data: RegisterForm): void => {
    if (data.username && data.password) {
      onRegister(data)
    }
  }
  return (
    <AuthLayout>
      <Container className={css.signUpContainer}>
        <Layout.Horizontal flex={{ alignItems: 'center' }}>
          <Text font={{ size: 'large', weight: 'bold' }} color={Color.BLACK}>
            {getString('signUp')}
          </Text>
          <FlexExpander />

          <Layout.Horizontal spacing="xsmall">
            <Text>{getString('alreadyHaveAccount')}</Text>
            <Link to={routes.toSignIn()}>{getString('signIn')}</Link>
          </Layout.Horizontal>
        </Layout.Horizontal>

        <Container margin={{ top: 'xlarge' }}>
          <Formik<RegisterForm>
            initialValues={{ username: '', email: '', password: '', confirmPassword: '' }}
            formName="loginPageForm"
            validationSchema={Yup.object().shape({
              username: Yup.string().required(getString('userNameRequired')),
              email: Yup.string().email().required(getString('emailRequired')),
              password: Yup.string().min(6, getString('minPassLimit')).required(getString('passwordRequired')),
              confirmPassword: Yup.string()
                .required(getString('confirmPassRequired'))
                .oneOf([Yup.ref('password')], getString('matchPassword'))
            })}
            onSubmit={handleSubmit}>
            <FormikForm>
              <FormInput.Text
                placeholder={getString('enterUser')}
                name="username"
                label={getString('userId')}
                disabled={false}
              />
              <FormInput.Text placeholder={'email@work.com'} name="email" label={getString('email')} disabled={false} />

              <FormInput.Text
                name="password"
                label={getString('password')}
                inputGroup={{ type: 'password' }}
                disabled={false}
                placeholder={getString('characterLimit')}
              />
              <FormInput.Text
                name="confirmPassword"
                label={getString('confirmPassword')}
                inputGroup={{ type: 'password' }}
                disabled={false}
                placeholder={getString('confirmPassword')}
              />

              <Button type="submit" intent="primary" loading={false} disabled={false} width="100%">
                {getString('signUp')}
              </Button>
            </FormikForm>
          </Formik>
        </Container>

        <Layout.Horizontal margin={{ top: 'xlarge' }} spacing="xsmall">
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
