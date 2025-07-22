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

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Container, FormikForm, Layout, Text, TextInput } from '@harnessio/uicore'
import { debounce, defaultTo, get, isEqual } from 'lodash-es'
import { useFormikContext } from 'formik'
import { GitFork, Repository } from 'iconoir-react'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import type { OpenapiCreateGitspaceRequest, TypesRepoResponse } from 'services/cde'
import { useListGitspaceBranches, useListGitspaceRepos, useRepoLookupForGitspace } from 'services/cde'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import {
  onPremSCMOptions,
  type RepoQueryParams,
  scmOptions,
  SCMType
} from 'cde-gitness/pages/GitspaceCreate/CDECreateGitspace'
import { useQueryParams } from 'hooks/useQueryParams'
import { isValidUrl, getRepoIdFromURL } from 'cde-gitness/utils/SelectRepository.utils'
import { useAppContext } from 'AppContext'
import { GitspaceSelect } from '../GitspaceSelect/GitspaceSelect'
import css from './CDEAnyGitImport.module.scss'

enum RepoCheckStatus {
  Valid = 'valid',
  InValid = 'InValid'
}

export const CDEAnyGitImport = () => {
  const repoQueryParams = useQueryParams<RepoQueryParams>()
  const { hooks } = useAppContext()
  const { getRepoURLPromise, useGetPaginatedListOfReposByRefConnector, useGetPaginatedListOfBranchesByRefConnector } =
    hooks

  const { setValues, setFieldError, values } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '' } = useGetCDEAPIParams()

  const [searchTerm, setSearchTerm] = useState<string | undefined>(values?.code_repo_url as string)
  const [searchBranch, setSearchBranch] = useState<string | undefined>(values?.branch as string)

  const isOnPremSCM = onPremSCMOptions.find(item => item.value === values?.code_repo_type)

  const { mutate, loading } = useRepoLookupForGitspace({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier
  })

  const { data: repoData, loading: repoLoading } = useListGitspaceRepos({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    queryParams: {
      search_term: defaultTo(searchTerm, ''),
      repo_type: values?.code_repo_type as string
    },
    debounce: 1000,
    lazy: !!isOnPremSCM
  })

  const { data: scmrepos, loading: onPremRepoLoading } = useGetPaginatedListOfReposByRefConnector({
    queryParams: {
      accountIdentifier,
      orgIdentifier,
      projectIdentifier,
      useSCMProviderForConnector: true,
      scmProviderForConnectorType: values?.code_repo_type,
      repoNameSearchTerm: searchTerm?.split('/')[searchTerm?.split('/')?.length - 1]
    },
    lazy: !isOnPremSCM
  })

  const {
    data: scmreposbranches,
    loading: scmbranchLoading,
    refetch: scmrefetchBranch
  } = useGetPaginatedListOfBranchesByRefConnector({
    queryParams: {
      accountIdentifier,
      orgIdentifier,
      projectIdentifier,
      repoName: values.name || '',
      useSCMProviderForConnector: true,
      scmProviderForConnectorType: values?.code_repo_type,
      branchNameSearchTerm: defaultTo(searchBranch, '')
    },
    lazy: true
  })

  const {
    data: branchData,
    loading: branchLoading,
    refetch: refetchBranch
  } = useListGitspaceBranches({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    queryParams: {
      search_term: defaultTo(searchBranch, ''),
      repo_type: values.code_repo_type || '',
      repo_url: values.code_repo_url || ''
    },
    debounce: 1000,
    lazy: true
  })

  const branchOptions: { name: string }[] = isOnPremSCM
    ? scmreposbranches?.data?.gitBranchesResponse?.branches
    : branchData?.branches || []

  const [repoCheckState, setRepoCheckState] = useState<RepoCheckStatus | undefined>()
  const [repoOptions, setRepoOptions] = useState<TypesRepoResponse[] | null | undefined>(
    isOnPremSCM ? scmrepos?.data?.gitRepositoryResponseList : repoData?.repositories
  )

  useEffect(() => {
    if (values?.code_repo_type) {
      setRepoCheckState(undefined)
      setSearchTerm(undefined)
      setSearchBranch(undefined)
      setRepoOptions(undefined)
    }
  }, [values?.code_repo_type])

  useEffect(() => {
    if (
      !isEqual(repoOptions, repoData?.repositories) ||
      !isEqual(repoOptions, scmrepos?.data?.gitRepositoryResponseList)
    ) {
      setRepoOptions(isOnPremSCM ? scmrepos?.data?.gitRepositoryResponseList : repoData?.repositories)
    }
  }, [repoOptions, repoData?.repositories, scmrepos])

  useEffect(() => {
    if (values.code_repo_url === repoQueryParams.codeRepoURL && repoQueryParams.codeRepoURL) {
      onChange(repoQueryParams.codeRepoURL as string, Boolean(repoQueryParams.branch))
    }
  }, [values.code_repo_url, repoQueryParams.codeRepoURL])

  useEffect(() => {
    if (searchBranch && !isOnPremSCM) {
      refetchBranch()
    }
  }, [searchBranch])

  useEffect(() => {
    if (isOnPremSCM) {
      scmrefetchBranch()
    }
  }, [values?.code_repo_url])

  const onChange = useCallback(
    debounce(async (url: string, skipBranchUpdate?: boolean) => {
      let errorMessage = ''
      try {
        if (isValidUrl(url)) {
          if (!isOnPremSCM) {
            const response = (await mutate({ url, repo_type: values?.code_repo_type })) as {
              is_private?: boolean
              branch: string
              url: string
            }
            const branchValue = skipBranchUpdate ? {} : { branch: response.branch }
            setValues((prvValues: any) => {
              return {
                ...prvValues,
                code_repo_url: response.url,
                ...branchValue,
                identifier: getRepoIdFromURL(response.url),
                name: '',
                code_repo_type: values?.code_repo_type
              }
            })
            if (!skipBranchUpdate) {
              setSearchBranch(response.branch)
            }
            setRepoCheckState(RepoCheckStatus.Valid)
          } else {
            const response = await getRepoURLPromise({
              queryParams: {
                accountIdentifier,
                orgIdentifier,
                projectIdentifier,
                useSCMProviderForConnector: true,
                repoName: url || '',
                scmProviderForConnectorType: values?.code_repo_type
              }
            })

            const branchValue = skipBranchUpdate ? {} : { branch: response.branch }
            setValues((prvValues: any) => {
              return {
                ...prvValues,
                code_repo_url: url,
                ...branchValue,
                identifier: getRepoIdFromURL(url),
                name: '',
                code_repo_type: values?.code_repo_type
              }
            })

            setRepoCheckState(RepoCheckStatus.Valid)
          }
        }
      } catch (err) {
        errorMessage = get(err, 'message') || ''
      }
      setFieldError('code_repo_url', errorMessage)
    }, 1000),
    [repoCheckState, values?.code_repo_type]
  )

  const branchRef = useRef<HTMLInputElement | null | undefined>()
  const repoRef = useRef<HTMLInputElement | null | undefined>()

  const scmOption = [...scmOptions, ...onPremSCMOptions].find(item => item.value === values.code_repo_type) as SCMType

  return (
    <FormikForm>
      <Layout.Horizontal spacing="medium">
        <Container width="63%" className={css.formFields}>
          <GitspaceSelect
            hideMenu={isValidUrl(defaultTo(searchTerm, ''))}
            text={
              <Container flex={{ alignItems: 'center' }} className={css.customTextInput}>
                <Repository height={32} width={32} />
                <TextInput
                  inputRef={ref => (repoRef.current = ref)}
                  value={searchTerm}
                  placeholder="enter url or type repo name"
                  onChange={async event => {
                    const target = event.target as HTMLInputElement
                    setSearchTerm(target?.value?.trim() || '')
                    await onChange(target.value)
                  }}
                />
              </Container>
            }
            tooltipProps={{ isOpen: repoRef.current?.onfocus }}
            rightIcon={
              loading || repoLoading || onPremRepoLoading
                ? 'loading'
                : repoCheckState && isValidUrl(defaultTo(searchTerm, ''))
                ? repoCheckState === RepoCheckStatus.Valid
                  ? 'tick-circle'
                  : 'warning-sign'
                : 'chevron-down'
            }
            withoutCurrentColor
            formikName="code_repo_url"
            renderMenu={
              <Menu style={{ maxHeight: 300 }}>
                {loading || repoLoading || onPremRepoLoading ? (
                  <MenuItem text={<Text>Fetching Repositories</Text>} />
                ) : repoOptions?.length ? (
                  repoOptions?.map(item => {
                    return (
                      <MenuItem
                        key={item.name}
                        disabled={repoLoading || onPremRepoLoading}
                        text={
                          <Layout.Horizontal
                            spacing="large"
                            flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
                            <img
                              height={26}
                              width={26}
                              src={defaultTo(scmOption?.icon, '')}
                              style={{ marginRight: '10px' }}
                            />
                            <Layout.Vertical>
                              <Text color={Color.BLACK}>{item.name}</Text>
                              <Text font={{ size: 'small' }}>{item.clone_url}</Text>
                            </Layout.Vertical>
                          </Layout.Horizontal>
                        }
                        onClick={async () => {
                          setSearchTerm(item.name as string)
                          if (isOnPremSCM) {
                            const { data } = await getRepoURLPromise({
                              queryParams: {
                                accountIdentifier,
                                orgIdentifier,
                                projectIdentifier,
                                useSCMProviderForConnector: true,
                                repoName: item.name || '',
                                scmProviderForConnectorType: values?.code_repo_type
                              }
                            })
                            setValues((prvValues: any) => {
                              return {
                                ...prvValues,
                                code_repo_url: data,
                                branch: item.default_branch,
                                identifier: getRepoIdFromURL(item.name),
                                name: '',
                                code_repo_type: values?.code_repo_type
                              }
                            })
                            // scmrefetchBranch()
                            setSearchBranch(undefined)
                          } else {
                            setValues((prvValues: any) => {
                              return {
                                ...prvValues,
                                code_repo_url: item.clone_url,
                                branch: item.default_branch,
                                identifier: getRepoIdFromURL(item.clone_url),
                                name: '',
                                code_repo_type: values?.code_repo_type
                              }
                            })
                            setSearchBranch(item.default_branch as string)
                          }

                          if (!isOnPremSCM) {
                            refetchBranch()
                          }
                        }}
                      />
                    )
                  })
                ) : (
                  <MenuItem text={<Text>No Repositories Found</Text>} />
                )}
              </Menu>
            }
          />
        </Container>
        <Container width="35%" className={css.formFields}>
          <GitspaceSelect
            text={
              <Container flex={{ alignItems: 'center' }} className={css.customTextInput}>
                <GitFork height={32} width={32} />
                <TextInput
                  inputRef={ref => (branchRef.current = ref)}
                  value={searchBranch}
                  placeholder="enter branch name"
                  onChange={async event => {
                    const target = event.target as HTMLInputElement
                    setSearchBranch(target?.value?.trim() || '')
                    setValues((prvValues: any) => {
                      return {
                        ...prvValues,
                        branch: target?.value?.trim() || ''
                      }
                    })
                  }}
                />
              </Container>
            }
            tooltipProps={{ isOpen: branchRef.current?.onfocus }}
            rightIcon={loading || branchLoading || scmbranchLoading ? 'loading' : 'chevron-down'}
            withoutCurrentColor
            formikName="branch"
            renderMenu={
              <Menu>
                {branchOptions?.length ? (
                  branchOptions?.map(item => {
                    return (
                      <MenuItem
                        key={item.name}
                        text={<Text>{item.name}</Text>}
                        onClick={() => {
                          setSearchBranch(item.name as string)
                          setValues((prvValues: any) => {
                            return {
                              ...prvValues,
                              branch: item.name
                            }
                          })
                        }}
                      />
                    )
                  })
                ) : loading || repoLoading || onPremRepoLoading || scmbranchLoading ? (
                  <MenuItem text={<Text>Fetching Branches</Text>} />
                ) : (
                  <MenuItem text={<Text>No Branches Found</Text>} />
                )}
              </Menu>
            }
          />
        </Container>
      </Layout.Horizontal>
    </FormikForm>
  )
}
