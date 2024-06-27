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
import { useHistory, useParams } from 'react-router-dom'
import { omit } from 'lodash-es'
import { useCreateGitspace, OpenapiCreateGitspaceRequest, useGetGitspace, useUpdateGitspace } from 'services/cde'
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
  ide: IDEType.VSCODEWEB
}

const GitspaceForm = () => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()

  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const { mutate, loading, error } = useCreateGitspace({
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  const { data: gitspaceData, loading: loadingGitspace } = useGetGitspace({
    gitspaceIdentifier: gitspaceId,
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  const { mutate: updateGitspace, loading: updatingGitspace } = useUpdateGitspace({
    gitspaceIdentifier: gitspaceId,
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  const formInitialData = gitspaceId ? { ...gitspaceData?.config } : initData

  if (error) {
    showError(getErrorMessage(error))
  }

  return (
    <>
      <Text className={css.cardTitle} font={{ variation: FontVariation.CARD_TITLE }}>
        {gitspaceId
          ? `${getString('cde.editGitspace')}  ${gitspaceData?.config?.name}`
          : getString('cde.createGitspace')}
      </Text>
      <Formik<OpenapiCreateGitspaceRequest & { validated?: boolean }>
        onSubmit={async data => {
          try {
            if (gitspaceId) {
              await updateGitspace(omit(data, 'metadata'))
              showSuccess(getString('cde.gitspaceUpdateSuccess'))
              history.push(`${routes.toCDEGitspaces({ space })}`)
            } else {
              const createdGitspace = await mutate(omit(data, 'metadata'))
              history.push(
                `${routes.toCDEGitspaceDetail({
                  space,
                  gitspaceId: createdGitspace?.config?.id || ''
                })}?redirectFrom=login`
              )
            }
          } catch (err) {
            showError(getErrorMessage(err))
          }
        }}
        formLoading={loadingGitspace || updatingGitspace}
        enableReinitialize
        formName={'createGitSpace'}
        initialValues={{ ...formInitialData, validated: false }}
        validateOnMount={false}
        validationSchema={yup.object().shape({
          branch: yup.string().trim().required(getString('cde.branchValidationMessage')),
          code_repo_type: yup.string().trim().required(getString('cde.repoValidationMessage')),
          code_repo_url: yup.string().trim().required(getString('cde.repoValidationMessage')),
          id: yup.string().trim().required(),
          ide: yup.string().trim().required(),
          infra_provider_resource_id: yup.string().trim().required(getString('cde.machineValidationMessage')),
          name: yup.string().trim().required(),
          metadata: yup.object().shape({
            region: yup.string().trim().required(getString('cde.regionValidationMessage'))
          })
        })}>
        {_ => {
          return (
            <FormikForm>
              <Layout.Vertical spacing="medium">
                <Layout.Horizontal spacing="medium">
                  <SelectRepository disabled={!!gitspaceId} />
                  <BranchInput disabled={!!gitspaceId} />
                </Layout.Horizontal>
                <SelectIDE />
                <SelectInfraProvider />
                <Button variation={ButtonVariation.PRIMARY} height={50} type="submit" loading={loading}>
                  {gitspaceId ? getString('cde.updateGitspace') : getString('cde.createGitspace')}
                </Button>
              </Layout.Vertical>
            </FormikForm>
          )
        }}
      </Formik>
    </>
  )
}

export const CreateGitspace = () => {
  return (
    <Card className={css.main}>
      <GitspaceForm />
    </Card>
  )
}
