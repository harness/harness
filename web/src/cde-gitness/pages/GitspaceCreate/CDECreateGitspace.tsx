/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import React, { useEffect, useState } from 'react'
import { Button, ButtonVariation, Container, Formik, FormikForm, Layout, Text, useToaster } from '@harnessio/uicore'
import { useHistory } from 'react-router-dom'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, MenuItem } from '@blueprintjs/core'
import { defaultTo, omit } from 'lodash-es'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { GitspaceSelect } from 'cde-gitness/components/GitspaceSelect/GitspaceSelect'
import harnessCode from 'cde-gitness/assests/harnessCode.svg?url'
import genericGit from 'cde-gitness/assests/genericGit.svg?url'
import gitnessIcon from 'cde-gitness/assests/gitness.svg?url'
import github from 'cde-gitness/assests/github.svg?url'
import gitlab from 'cde-gitness/assests/gitlab.svg?url'
import bitbucket from 'cde-gitness/assests/bitbucket.svg?url'
import { CDEAnyGitImport } from 'cde-gitness/components/CDEAnyGitImport/CDEAnyGitImport'
import { CDEIDESelect } from 'cde-gitness/components/CDEIDESelect/CDEIDESelect'
import { SelectInfraProvider } from 'cde-gitness/components/SelectInfraProvider/SelectInfraProvider'
import { OpenapiCreateGitspaceRequest, useCreateGitspace } from 'services/cde'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { EnumGitspaceCodeRepoType, IDEType } from 'cde-gitness/constants'
import { CDESSHSelect } from 'cde-gitness/components/CDESSHSelect/CDESSHSelect'
import { useQueryParams } from 'hooks/useQueryParams'
import { getRepoIdFromURL, getRepoNameFromURL } from 'cde-gitness/components/CDEAnyGitImport/CDEAnyGitImport.utils'
import { CDEUnknownSCM } from 'cde-gitness/components/CDEAnyGitImport/CDEUnknownSCM'
import { gitnessFormInitialValues } from './GitspaceCreate.constants'
import { validateGitnessForm } from './GitspaceCreate.utils'
import css from './GitspaceCreate.module.scss'

export interface SCMType {
  name: string
  value: EnumGitspaceCodeRepoType
  icon: string
}

export interface RepoQueryParams {
  name?: string
  identifier?: string
  branch?: string
  codeRepoURL?: string
  codeRepoType?: EnumGitspaceCodeRepoType
}

export const scmOptions: SCMType[] = [
  { name: 'Harness Code', value: EnumGitspaceCodeRepoType.HARNESS_CODE, icon: harnessCode },
  { name: 'GitHub Cloud', value: EnumGitspaceCodeRepoType.GITHUB, icon: github },
  { name: 'GitLab Cloud', value: EnumGitspaceCodeRepoType.GITLAB, icon: gitlab },
  { name: 'Bitbucket', value: EnumGitspaceCodeRepoType.BITBUCKET, icon: bitbucket },
  { name: 'Any public Git repository', value: EnumGitspaceCodeRepoType.UNKNOWN, icon: genericGit },
  { name: 'Gitness', value: EnumGitspaceCodeRepoType.GITNESS, icon: gitnessIcon }
]
export const scmOptionsCDE: SCMType[] = [
  { name: 'Harness Code', value: EnumGitspaceCodeRepoType.HARNESS_CODE, icon: harnessCode },
  { name: 'GitHub Cloud', value: EnumGitspaceCodeRepoType.GITHUB, icon: github },
  { name: 'GitLab Cloud', value: EnumGitspaceCodeRepoType.GITLAB, icon: gitlab },
  { name: 'Bitbucket', value: EnumGitspaceCodeRepoType.BITBUCKET, icon: bitbucket },
  { name: 'Any public Git repository', value: EnumGitspaceCodeRepoType.UNKNOWN, icon: genericGit }
]

export const CDECreateGitspace = () => {
  const { getString } = useStrings()
  const { routes, currentUserProfileURL, hooks, currentUser } = useAppContext()
  const { useGetUserSourceCodeManagers } = hooks
  const history = useHistory()
  const space = useGetSpaceParam()
  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '' } = useGetCDEAPIParams()
  const { showSuccess, showError } = useToaster()
  const { mutate } = useCreateGitspace({ accountIdentifier, orgIdentifier, projectIdentifier })
  const repoQueryParams = useQueryParams<RepoQueryParams>()

  const [repoURLviaQueryParam, setrepoURLviaQueryParam] = useState<RepoQueryParams>({ ...repoQueryParams })

  const { data: OauthSCMs } = useGetUserSourceCodeManagers({
    queryParams: { accountIdentifier, userIdentifier: currentUser?.uid }
  })

  useEffect(() => {
    const { codeRepoURL, codeRepoType, branch: queryParamBranch } = repoQueryParams
    if (codeRepoURL !== repoURLviaQueryParam.codeRepoURL && codeRepoType !== repoURLviaQueryParam.codeRepoType) {
      setrepoURLviaQueryParam(prv => {
        return {
          ...prv,
          name: getRepoNameFromURL(codeRepoURL),
          identifier: getRepoIdFromURL(codeRepoURL),
          branch: queryParamBranch,
          codeRepoURL,
          codeRepoType
        }
      })
    }
  }, [repoQueryParams])

  const oauthSCMsListTypes =
    OauthSCMs?.data?.userSourceCodeManagerResponseDTOList?.map((item: { type: string }) => item.type.toLowerCase()) ||
    []

  const includeQueryParams =
    repoQueryParams?.codeRepoURL && repoQueryParams?.codeRepoType
      ? {
          code_repo_url: repoQueryParams.codeRepoURL,
          branch: repoQueryParams.branch,
          identifier: getRepoIdFromURL(repoQueryParams.codeRepoURL),
          name: getRepoNameFromURL(repoQueryParams.codeRepoURL),
          code_repo_type: repoQueryParams.codeRepoType
        }
      : {}

  return (
    <Formik
      onSubmit={async data => {
        try {
          const payload = data
          const response = await mutate({
            ...omit(payload, 'metadata'),
            space_ref: space
          } as OpenapiCreateGitspaceRequest & {
            space_ref?: string
          })
          showSuccess(getString('cde.create.gitspaceCreateSuccess'))
          history.push(
            `${routes.toCDEGitspaceDetail({
              space,
              gitspaceId: response.identifier || ''
            })}?redirectFrom=login`
          )
        } catch (error) {
          showError(getString('cde.create.gitspaceCreateFailed'))
          showError(getErrorMessage(error))
        }
      }}
      initialValues={{
        ...gitnessFormInitialValues,
        code_repo_type: EnumGitspaceCodeRepoType.HARNESS_CODE,
        ...includeQueryParams
      }}
      validationSchema={validateGitnessForm(getString)}
      formName="importRepoForm"
      enableReinitialize>
      {formik => {
        const scmOption = scmOptionsCDE.find(item => item.value === formik.values.code_repo_type) as SCMType
        return (
          <>
            <Layout.Horizontal
              className={css.formTitleContainer}
              flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
              <Text font={{ variation: FontVariation.CARD_TITLE }}>{getString('cde.create.repositoryDetails')}</Text>
            </Layout.Horizontal>
            <FormikForm>
              <Container className={css.formContainer}>
                <Container>
                  <GitspaceSelect
                    formikName="code_repo_type"
                    text={
                      <Layout.Horizontal spacing="large" flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
                        <img
                          height={32}
                          width={32}
                          src={defaultTo(scmOption?.icon, '')}
                          style={{ marginRight: '10px' }}
                        />
                        <Layout.Vertical>
                          <Text font={{ variation: FontVariation.SMALL }}>Git Provider</Text>
                          <Text>{defaultTo(scmOption?.name, '')}</Text>
                        </Layout.Vertical>
                      </Layout.Horizontal>
                    }
                    renderMenu={
                      <Menu>
                        {scmOptionsCDE.map(item => (
                          <MenuItem
                            active={item.name === scmOption.name}
                            key={item.name}
                            text={
                              <Layout.Horizontal
                                spacing="large"
                                flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
                                <img height={24} width={24} src={item.icon} />
                                <Text>{item.name}</Text>
                              </Layout.Horizontal>
                            }
                            onClick={() => {
                              formik.setValues((prvValues: any) => {
                                return {
                                  ...prvValues,
                                  code_repo_url: undefined,
                                  branch: undefined,
                                  identifier: undefined,
                                  name: undefined,
                                  code_repo_type: item.value
                                }
                              })
                            }}
                          />
                        ))}
                      </Menu>
                    }
                  />
                  {![
                    EnumGitspaceCodeRepoType.HARNESS_CODE,
                    EnumGitspaceCodeRepoType.UNKNOWN,
                    ...oauthSCMsListTypes
                  ].includes(scmOption.value) ? (
                    <Layout.Vertical spacing="large">
                      <Container padding="medium" background={Color.YELLOW_100} border={{ color: Color.YELLOW_400 }}>
                        <Layout.Vertical spacing="large">
                          <Text>Please Configure the GitHub OAuth to connect to the repositories you have access</Text>
                          <Button
                            width="250px"
                            variation={ButtonVariation.PRIMARY}
                            onClick={() => {
                              history.push(currentUserProfileURL)
                            }}>
                            Configure the OAuth in Profile
                          </Button>
                          <Container>
                            <ol style={{ paddingLeft: '16px' }}>
                              <li>
                                <Text>Visit the User Profile Settings.</Text>
                              </li>
                              <li>
                                <Text>Under OAuth section, select Github and connect</Text>
                              </li>
                              <li>
                                <Text>
                                  After configuring return back to this page to connect the private repositories
                                </Text>
                              </li>
                            </ol>
                          </Container>
                        </Layout.Vertical>
                      </Container>
                      <CDEAnyGitImport />
                    </Layout.Vertical>
                  ) : scmOption.value === EnumGitspaceCodeRepoType.UNKNOWN ? (
                    <CDEUnknownSCM />
                  ) : (
                    <CDEAnyGitImport />
                  )}
                </Container>
              </Container>
              <Container className={css.formOuterContainer}>
                <CDEIDESelect onChange={formik.setFieldValue} selectedIde={formik.values.ide} />
                {formik.values.ide === IDEType.VSCODE && <CDESSHSelect />}
                <SelectInfraProvider />
                <Button width={'100%'} variation={ButtonVariation.PRIMARY} height={50} type="submit">
                  {getString('cde.createGitspace')}
                </Button>
              </Container>
            </FormikForm>
          </>
        )
      }}
    </Formik>
  )
}
