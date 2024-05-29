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
import { Button, ButtonVariation, Card, Formik, FormikForm, Layout, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useCreateGitspace, type EnumCodeRepoType, type EnumIDEType } from 'services/cde'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useStrings } from 'framework/strings'
import { IDEType } from 'cde/constants'
import { SelectIDE } from './components/SelectIDE/SelectIDE'
import { SelectRepository } from './components/SelectRepository/SelectRepository'
import { BranchInput } from './components/BranchInput/BranchInput'
import { SelectInfraProvider } from './components/SelectInfraProvider/SelectInfraProvider'
import css from './CreateGitspace.module.scss'

const initData = {
  ide: IDEType.VSCODE
}

export interface GitspaceFormInterface {
  ide?: EnumIDEType
  branch?: string
  codeRepoId?: string
  codeRepoUrl?: string
  codeRepoType?: EnumCodeRepoType
  region?: string
  infra_provider_resource_id?: string
}

interface GitspaceFormProps {
  onSubmit: (data: GitspaceFormInterface) => void
}

const GitspaceForm = ({ onSubmit }: GitspaceFormProps) => {
  const { getString } = useStrings()
  return (
    <Formik<GitspaceFormInterface>
      onSubmit={async data => await onSubmit(data)}
      formName={'createGitSpace'}
      initialValues={initData}
      validationSchema={yup.object().shape({
        codeRepoId: yup.string().trim().required(),
        branch: yup.string().trim().required(),
        ide: yup.string().trim().required(),
        region: yup.string().trim().required(),
        infra_provider_resource_id: yup.string().trim().required()
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
              <Button variation={ButtonVariation.PRIMARY} height={50} type="submit">
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
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { mutate } = useCreateGitspace({
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  return (
    <Card className={css.main}>
      <Text className={css.cardTitle} font={{ variation: FontVariation.CARD_TITLE }}>
        {getString('cde.createGitspace')}
      </Text>
      <GitspaceForm onSubmit={mutate} />
    </Card>
  )
}
