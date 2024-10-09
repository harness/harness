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
import { debounce, defaultTo, get } from 'lodash-es'
import { useFormikContext } from 'formik'
import { GitFork, Repository } from 'iconoir-react'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { OpenapiCreateGitspaceRequest } from 'services/cde'
import { useListGitspaceRepos, useListGitspaceBranches, useRepoLookupForGitspace } from 'services/cde'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { scmOptions, SCMType, type RepoQueryParams } from 'cde-gitness/pages/GitspaceCreate/CDECreateGitspace'
import { useQueryParams } from 'hooks/useQueryParams'
import { getRepoIdFromURL, getRepoNameFromURL, isValidUrl } from 'cde-gitness/utils/SelectRepository.utils'
import { GitspaceSelect } from '../GitspaceSelect/GitspaceSelect'
import css from './CDEAnyGitImport.module.scss'

enum RepoCheckStatus {
  Valid = 'valid',
  InValid = 'InValid'
}

export const CDEAnyGitImport = () => {
  const { getString } = useStrings()
  const repoQueryParams = useQueryParams<RepoQueryParams>()

  const { setValues, setFieldError, values } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '' } = useGetCDEAPIParams()

  const [searchTerm, setSearchTerm] = useState(values?.code_repo_url as string)
  const [searchBranch, setSearchBranch] = useState(values?.branch as string)

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
      search_term: searchTerm,
      repo_type: values?.code_repo_type as string
    },
    debounce: 1000
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
      search_term: searchBranch,
      repo_type: values.code_repo_type || '',
      repo_url: values.code_repo_url || ''
    },
    debounce: 1000,
    lazy: true
  })

  const [repoCheckState, setRepoCheckState] = useState<RepoCheckStatus | undefined>()

  useEffect(() => {
    if (values?.code_repo_type) {
      setRepoCheckState(undefined)
    }
  }, [values?.code_repo_type])

  useEffect(() => {
    if (values.code_repo_url === repoQueryParams.codeRepoURL && repoQueryParams.codeRepoURL) {
      onChange(repoQueryParams.codeRepoURL as string, Boolean(repoQueryParams.branch))
    }
  }, [values.code_repo_url, repoQueryParams.codeRepoURL])

  useEffect(() => {
    if (searchBranch) {
      refetchBranch()
    }
  }, [searchBranch])

  const onChange = useCallback(
    debounce(async (url: string, skipBranchUpdate?: boolean) => {
      let errorMessage = ''
      try {
        if (isValidUrl(url)) {
          const response = (await mutate({ url, repo_type: values?.code_repo_type })) as {
            is_private?: boolean
            branch: string
            url: string
          }
          if (!response?.branch) {
            errorMessage = getString('cde.repository.privateRepoWarning')
            setRepoCheckState(RepoCheckStatus.InValid)
            setValues((prvValues: any) => {
              return {
                ...prvValues,
                code_repo_url: response.url,
                branch: undefined,
                identifier: undefined,
                name: undefined,
                code_repo_type: values?.code_repo_type
              }
            })
            setTimeout(() => {
              setFieldError('code_repo_url', errorMessage)
            }, 500)
          } else {
            const branchValue = skipBranchUpdate ? {} : { branch: response.branch }
            setValues((prvValues: any) => {
              return {
                ...prvValues,
                code_repo_url: response.url,
                ...branchValue,
                identifier: getRepoIdFromURL(response.url),
                name: getRepoNameFromURL(response.url),
                code_repo_type: values?.code_repo_type
              }
            })
            if (!skipBranchUpdate) {
              setSearchBranch(response.branch)
            }
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

  const scmOption = scmOptions.find(item => item.value === values.code_repo_type) as SCMType

  return (
    <FormikForm>
      <Layout.Horizontal spacing="medium">
        <Container width="63%" className={css.formFields}>
          <GitspaceSelect
            hideMenu={isValidUrl(searchTerm)}
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
              loading || repoLoading
                ? 'loading'
                : repoCheckState && isValidUrl(searchTerm)
                ? repoCheckState === RepoCheckStatus.Valid
                  ? 'tick-circle'
                  : 'warning-sign'
                : 'chevron-down'
            }
            withoutCurrentColor
            formikName="code_repo_url"
            renderMenu={
              <Menu>
                {repoData?.repositories?.length ? (
                  repoData?.repositories?.map(item => {
                    return (
                      <MenuItem
                        key={item.name}
                        disabled={repoLoading}
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
                        onClick={() => {
                          setSearchTerm(item.name as string)
                          setValues((prvValues: any) => {
                            return {
                              ...prvValues,
                              code_repo_url: item.clone_url,
                              branch: item.default_branch,
                              identifier: getRepoIdFromURL(item.clone_url),
                              name: getRepoNameFromURL(item.clone_url),
                              code_repo_type: values?.code_repo_type
                            }
                          })
                          setSearchBranch(item.default_branch as string)
                          refetchBranch()
                        }}
                      />
                    )
                  })
                ) : loading || repoLoading ? (
                  <MenuItem text={<Text>Fetching Repositories</Text>} />
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
                  }}
                />
              </Container>
            }
            tooltipProps={{ isOpen: branchRef.current?.onfocus }}
            rightIcon={loading || branchLoading ? 'loading' : 'chevron-down'}
            withoutCurrentColor
            formikName="branch"
            renderMenu={
              <Menu>
                {(branchData as unknown as { branches: { name: string }[] })?.branches?.length ? (
                  (branchData as unknown as { branches: { name: string }[] })?.branches?.map(item => {
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
                ) : loading || repoLoading ? (
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
