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

import React from 'react'
import * as yup from 'yup'
import { Button, ButtonVariation, Card, Formik, FormikForm, Layout, Text, useToaster } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useCreateGitspace, OpenapiCreateGitspaceRequest } from 'services/cde'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useStrings } from 'framework/strings'
import { IDEType } from 'cde/constants'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { SelectIDE } from './components/SelectIDE/SelectIDE'
import { SelectRepository } from './components/SelectRepository/SelectRepository'
import { BranchInput } from './components/BranchInput/BranchInput'
import { SelectInfraProvider } from './components/SelectInfraProvider/SelectInfraProvider'
import css from './CreateGitspace.module.scss'

const initData = {
  ide: IDEType.VSCODE
}

const GitspaceForm = () => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const { showError } = useToaster()
  const history = useHistory()

  const { mutate, loading, error } = useCreateGitspace({
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  if (error) {
    showError(getErrorMessage(error))
  }

  return (
    <Formik<OpenapiCreateGitspaceRequest>
      onSubmit={async data => {
        try {
          const createdGitspace = await mutate(data)
          history.push(
            `${routes.toCDEGitspaceDetail({
              space,
              gitspaceId: createdGitspace?.id || ''
            })}?redirectFrom=login`
          )
        } catch (err) {
          showError(getErrorMessage(err))
        }
      }}
      formName={'createGitSpace'}
      initialValues={initData}
      validateOnMount={false}
      validationSchema={yup.object().shape({
        branch: yup.string().trim().required(),
        code_repo_id: yup.string().trim().required(),
        code_repo_type: yup.string().trim().required(),
        code_repo_url: yup.string().trim().required(),
        id: yup.string().trim().required(),
        ide: yup.string().trim().required(),
        infra_provider_resource_id: yup.string().trim().required(),
        name: yup.string().trim().required(),
        metadata: yup.object().shape({
          region: yup.string().trim().required()
        })
      })}>
      {_ => {
        return (
          <FormikForm>
            <Layout.Vertical spacing="medium">
              <Layout.Horizontal spacing="medium">
                <SelectRepository />
                <BranchInput />
              </Layout.Horizontal>
              <SelectIDE />
              <SelectInfraProvider />
              <Button variation={ButtonVariation.PRIMARY} height={50} type="submit" loading={loading}>
                {getString('cde.createGitspace')}
              </Button>
            </Layout.Vertical>
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export const CreateGitspace = () => {
  const { getString } = useStrings()

  return (
    <Card className={css.main}>
      <Text className={css.cardTitle} font={{ variation: FontVariation.CARD_TITLE }}>
        {getString('cde.createGitspace')}
      </Text>
      <GitspaceForm />
    </Card>
  )
}
