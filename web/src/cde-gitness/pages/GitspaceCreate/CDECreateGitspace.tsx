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
import React, { useEffect, useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  Formik,
  FormikForm,
  FormInput,
  Layout,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { useHistory } from 'react-router-dom'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, MenuItem } from '@blueprintjs/core'
import { defaultTo, omit } from 'lodash-es'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { useFindGitspaceSettings } from 'services/cde'
import { GitspaceSelect } from 'cde-gitness/components/GitspaceSelect/GitspaceSelect'
import harnessCode from 'cde-gitness/assests/harnessCode.svg?url'
import codeSandboxIcon from 'cde-gitness/assests/codeSandboxLogo.svg?url'
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
import { EnumGitspaceCodeRepoType, getIDETypeOptions, getIDEOption } from 'cde-gitness/constants'
import { CDESSHSelect } from 'cde-gitness/components/CDESSHSelect/CDESSHSelect'
import { useQueryParams } from 'hooks/useQueryParams'
import { CDEUnknownSCM } from 'cde-gitness/components/CDEAnyGitImport/CDEUnknownSCM'
import { useFilteredIdeOptions } from 'cde-gitness/hooks/useFilteredIdeOptions'
import { gitnessFormInitialValues } from './GitspaceCreate.constants'
import { validateGitnessForm } from './GitspaceCreate.utils'
import { generateGitspaceName, getIdentifierFromName } from '../../utils/nameGenerator.utils'
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

export const onPremSCMOptions: SCMType[] = [
  { name: 'GitHub Enterprise', value: EnumGitspaceCodeRepoType.GITHUB_ENTERPRISE, icon: github },
  { name: 'GitLab On-prem', value: EnumGitspaceCodeRepoType.GITLAB_ON_PREM, icon: gitlab },
  { name: 'Bitbucket Server', value: EnumGitspaceCodeRepoType.BITBUCKET_SERVER, icon: bitbucket },
  { name: 'Any public Git repository', value: EnumGitspaceCodeRepoType.UNKNOWN, icon: genericGit }
]

export const scmOptionsCDE: SCMType[] = [
  { name: 'Harness Code', value: EnumGitspaceCodeRepoType.HARNESS_CODE, icon: harnessCode },
  { name: 'GitHub Cloud', value: EnumGitspaceCodeRepoType.GITHUB, icon: github },
  { name: 'GitLab Cloud', value: EnumGitspaceCodeRepoType.GITLAB, icon: gitlab },
  { name: 'Bitbucket', value: EnumGitspaceCodeRepoType.BITBUCKET, icon: bitbucket },
  ...onPremSCMOptions
]

export const CDECreateGitspace = () => {
  const { getString } = useStrings()
  const { routes, currentUserProfileURL, hooks, currentUser } = useAppContext()
  const { useGetUserSourceCodeManagers } = hooks
  const history = useHistory()
  const space = useGetSpaceParam()
  const [isGeneratingName, setIsGeneratingName] = useState(true)
  const [generatedName, setGeneratedName] = useState<string>('')
  const suggestedName = useMemo(() => generateGitspaceName(), [])

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '' } = useGetCDEAPIParams()
  const { showSuccess, showError } = useToaster()
  const { mutate } = useCreateGitspace({ accountIdentifier, orgIdentifier, projectIdentifier })
  const repoQueryParams = useQueryParams<RepoQueryParams>()
  const [filteredSCMOptions, setFilteredSCMOptions] = useState<SCMType[]>(scmOptionsCDE)

  const ideOptions = useMemo(() => getIDETypeOptions(getString) ?? [], [getString])

  const [repoURLviaQueryParam, setrepoURLviaQueryParam] = useState<RepoQueryParams>({ ...repoQueryParams })

  const { data: gitspaceSettings, error: settingsError } = useFindGitspaceSettings({
    accountIdentifier
  })

  useEffect(() => {
    if (settingsError) {
      showError(getErrorMessage(settingsError))
    }
  }, [settingsError, getString, showError])

  useEffect(() => {
    if (gitspaceSettings?.settings?.gitspace_config) {
      const { scm } = gitspaceSettings.settings.gitspace_config

      if (scm?.access_list?.mode === 'deny' && Array.isArray(scm.access_list.list)) {
        const scmDenyList = scm.access_list.list
        const filteredOptions = scmOptionsCDE.filter(
          option => !scmDenyList.includes(option.value as EnumGitspaceCodeRepoType)
        )
        setFilteredSCMOptions(filteredOptions)
      }
    }
  }, [gitspaceSettings])

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      setGeneratedName(suggestedName)
      setIsGeneratingName(false)
    }, 2000)
    return () => {
      clearTimeout(timeoutId)
    }
  }, [suggestedName])

  const filteredIdeOptions = useFilteredIdeOptions(ideOptions, gitspaceSettings, getString)

  const defaultIdeType = useMemo(() => {
    return filteredIdeOptions.length > 0 ? filteredIdeOptions[0].value : undefined
  }, [filteredIdeOptions])

  const { data: OauthSCMs } = useGetUserSourceCodeManagers({
    queryParams: { accountIdentifier, userIdentifier: currentUser?.uid }
  })

  useEffect(() => {
    const { codeRepoURL, codeRepoType, branch: queryParamBranch } = repoQueryParams
    if (codeRepoURL !== repoURLviaQueryParam.codeRepoURL && codeRepoType !== repoURLviaQueryParam.codeRepoType) {
      setrepoURLviaQueryParam(prv => {
        return {
          ...prv,
          name: prv.name,
          identifier: '',
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
          identifier: '',
          name: '',
          code_repo_type: repoQueryParams.codeRepoType
        }
      : {}

  return (
    <Formik
      onSubmit={async data => {
        try {
          const payload = { ...data, identifier: getIdentifierFromName(data.name) }
          const response = await mutate({
            ...omit(payload, 'metadata'),
            space_ref: space,
            infra_provider_config_identifier: data?.metadata?.infraProvider
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
        code_repo_type: filteredSCMOptions.length > 0 ? filteredSCMOptions[0].value : getString('cde.create.scmEmpty'),
        ide: defaultIdeType || getString('cde.create.ideEmpty'),
        ...includeQueryParams
      }}
      validateOnChange={true}
      validationSchema={validateGitnessForm(getString, true)}
      formName="importRepoForm"
      enableReinitialize>
      {formik => {
        const scmOption = formik.values?.code_repo_type
          ? filteredSCMOptions.find(item => item.value === formik.values.code_repo_type) ||
            scmOptionsCDE.find(item => item.value === formik.values.code_repo_type)
          : undefined
        const selectedIDE = formik?.values?.ide ? getIDEOption(formik?.values?.ide, getString) : null
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
                        {filteredSCMOptions.length === 0 ? (
                          <Layout.Vertical>
                            <Text font={{ variation: FontVariation.SMALL }}>{getString('cde.create.gitprovider')}</Text>
                            <Text>{getString('cde.create.scmEmpty')}</Text>
                          </Layout.Vertical>
                        ) : (
                          <>
                            <img
                              height={32}
                              width={32}
                              src={defaultTo(scmOption?.icon, '')}
                              style={{ marginRight: '10px' }}
                            />
                            <Layout.Vertical>
                              <Text font={{ variation: FontVariation.SMALL }}>
                                {getString('cde.create.gitprovider')}
                              </Text>
                              <Text>{defaultTo(scmOption?.name || {}, '')}</Text>
                            </Layout.Vertical>
                          </>
                        )}
                      </Layout.Horizontal>
                    }
                    renderMenu={
                      <Menu>
                        {filteredSCMOptions.map(item => (
                          <MenuItem
                            active={item.name === scmOption?.name}
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
                                  name: prvValues.name,
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
                  ].includes(scmOption?.value) ? (
                    <Layout.Vertical spacing="large">
                      <Container padding="medium" background={Color.YELLOW_100} border={{ color: Color.YELLOW_400 }}>
                        <Layout.Vertical spacing="large">
                          <Text>
                            {`Please Configure ${
                              scmOption?.name || 'your Git provider'
                            } OAuth to connect to the repositories you have access`}
                          </Text>
                          <Button
                            width="250px"
                            variation={ButtonVariation.PRIMARY}
                            onClick={() => {
                              history.push(currentUserProfileURL)
                            }}>
                            {getString('cde.create.githubOauthhelpertext2')}
                          </Button>
                          <Container>
                            <ol style={{ paddingLeft: '16px' }}>
                              <li>
                                <Text>{getString('cde.create.githubOauthhelpertext3')}</Text>
                              </li>
                              <li>
                                <Text>{`Under OAuth section, select ${
                                  scmOption?.name || 'your Git provider'
                                } and connect`}</Text>
                              </li>
                              <li>
                                <Text>{getString('cde.create.githubOauthhelpertext5')}</Text>
                              </li>
                            </ol>
                          </Container>
                        </Layout.Vertical>
                      </Container>
                      <CDEAnyGitImport />
                    </Layout.Vertical>
                  ) : scmOption?.value === EnumGitspaceCodeRepoType.UNKNOWN ? (
                    <CDEUnknownSCM />
                  ) : (
                    <CDEAnyGitImport />
                  )}
                </Container>
              </Container>

              <Container className={css.formOuterContainer}>
                <Layout.Horizontal className={css.gitspaceNameContainer}>
                  <Container width="69.5%">
                    <Layout.Horizontal className={css.leftSection}>
                      <img src={codeSandboxIcon} alt="gitspace" className={css.icon} />
                      <Layout.Vertical className={css.textSection} spacing={'small'}>
                        <Text color={Color.GREY_500} font={{ weight: 'bold' }}>
                          {getString('cde.create.gitspaceNameLabel')}
                        </Text>
                        <Layout.Vertical spacing={'xsmall'}>
                          <Text font={'small'}>{getString('cde.create.gitspaceNameHelpertext1')}</Text>
                          <Layout.Horizontal spacing={'xsmall'}>
                            <Text font={'small'}>{getString('cde.create.gitspaceNameHelpertext2')}</Text>
                            <Text
                              className={css.suggestedName}
                              font={'small'}
                              onClick={e => {
                                e.stopPropagation()
                                formik.setFieldValue('name', suggestedName)
                              }}>
                              {isGeneratingName ? <Icon name="loading" /> : generatedName}
                            </Text>
                          </Layout.Horizontal>
                        </Layout.Vertical>
                      </Layout.Vertical>
                    </Layout.Horizontal>
                  </Container>
                  <Container width="30.5%">
                    <FormInput.Text
                      name="name"
                      placeholder={getString('cde.create.gitspaceNamePlaceholder')}
                      className={css.inputFieldContainer}
                    />
                  </Container>
                </Layout.Horizontal>

                <CDEIDESelect
                  onChange={formik.setFieldValue}
                  selectedIde={formik.values.ide}
                  filteredIdeOptions={filteredIdeOptions}
                />
                {selectedIDE?.allowSSH ? <CDESSHSelect /> : <></>}
                <SelectInfraProvider />
                <Container style={{ display: 'flex', justifyContent: 'flex-end' }}>
                  <Button width={'20%'} variation={ButtonVariation.PRIMARY} height={50} type="submit">
                    {getString('cde.createGitspace')}
                  </Button>
                </Container>
              </Container>
            </FormikForm>
          </>
        )
      }}
    </Formik>
  )
}
