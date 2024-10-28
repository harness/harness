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
import { useGet } from 'restful-react'
import { Button, ButtonVariation, Container, ExpandingSearchInput, Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { Repository } from 'iconoir-react'
import { useFormikContext } from 'formik'
import type { RepoRepositoryOutput } from 'services/code'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { String, useStrings } from 'framework/strings'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import NewRepoModalButton from 'components/NewRepoModalButton/NewRepoModalButton'
import noRepo from 'cde-gitness/assests/noRepo.svg?url'
import { RepoCreationType } from 'utils/GitUtils'
import gitnessRepoLogo from 'cde-gitness/assests/gitness.svg?url'
import { EnumGitspaceCodeRepoType } from 'cde-gitness/constants'
import { useQueryParams } from 'hooks/useQueryParams'
import type { RepoQueryParams } from 'cde-gitness/pages/GitspaceCreate/CDECreateGitspace'
import { GitspaceSelect } from '../GitspaceSelect/GitspaceSelect'
import css from './GitnessRepoImportForm.module.scss'

const RepositoryText = ({
  repoList,
  value,
  isCDE
}: {
  repoList: RepoRepositoryOutput[] | null
  value?: string
  isCDE?: boolean
}) => {
  const { getString } = useStrings()
  const repoMetadata = repoList?.find(repo => repo.git_url === value)
  const repoName = repoMetadata?.path

  return (
    <Layout.Horizontal spacing={'medium'} flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
      {isCDE ? (
        <Repository color="none" height={24} width={24} />
      ) : (
        <img src={gitnessRepoLogo} height={24} width={24} />
      )}
      {repoName ? (
        <Container margin={{ left: 'medium' }}>
          <Layout.Vertical spacing="xsmall">
            <Text font={'normal'}>{getString('cde.repository.repo')}</Text>
            <Text color={Color.BLACK} font={'small'} lineClamp={1}>
              {repoName || getString('cde.repository.repositoryURL')}
            </Text>
          </Layout.Vertical>
        </Container>
      ) : (
        <Text font={'normal'}>{getString('cde.repository.selectRepository')}</Text>
      )}
    </Layout.Horizontal>
  )
}

const BranchText = ({ value }: { value?: string }) => {
  const { getString } = useStrings()
  return (
    <Layout.Horizontal spacing={'medium'} flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
      <Icon name={'git-branch'} size={24} />
      {value ? (
        <Container margin={{ left: 'medium' }}>
          <Layout.Vertical spacing="xsmall">
            <Text font={'normal'}>{getString('branch')}</Text>
            <Text color={Color.BLACK} font={'small'} lineClamp={1}>
              {value}
            </Text>
          </Layout.Vertical>
        </Container>
      ) : (
        <Text font={'normal'}>{getString('cde.create.selectBranchPlaceholder')}</Text>
      )}
    </Layout.Horizontal>
  )
}

export const GitnessRepoImportForm = ({ isCDE }: { isCDE?: boolean }) => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const [branchSearch, setBranchSearch] = useState('')
  const [repoSearch, setRepoSearch] = useState('')
  const [hadReops, setHadRepos] = useState(false)
  const [repoRef, setReporef] = useState('')

  const {
    data: repositories,
    loading,
    refetch: refetchRepos
  } = useGet<RepoRepositoryOutput[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    queryParams: { query: repoSearch },
    debounce: 500
  })

  useEffect(() => {
    if (!hadReops && repositories?.length) {
      setHadRepos(true)
    }
  }, [repositories])

  const {
    data: branches,
    refetch,
    loading: loadingBranches
  } = useGet<{ name: string }[]>({
    path: `/api/v1/repos/${repoRef}/+/branches`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page: 1,
      sort: 'date',
      order: 'desc',
      include_commit: false,
      query: branchSearch
    },
    lazy: true
  })

  useEffect(() => {
    if (repoRef || branchSearch) {
      refetch()
    }
  }, [repoRef, branchSearch])

  const repoListOptions = repositories || []
  const hideInitialMenu = Boolean(repoSearch) || Boolean(repositories)

  const repoQueryParams = useQueryParams<RepoQueryParams>()

  useEffect(() => {
    if (isCDE) {
      const repoData = repoListOptions?.find(repo => repo.git_url === repoQueryParams.codeRepoURL)
      if (!repoQueryParams.branch && repoData) {
        formik.setFieldValue('branch', repoData?.default_branch)
      }
    }
  }, [repoListOptions, isCDE])

  const formik = useFormikContext<any>()

  const { values } = formik
  const repoMetadata = repoListOptions.find(repo => repo.git_url === values.code_repo_url)
  if (repoRef !== repoMetadata?.path) {
    setReporef(repoMetadata?.path as string)
  }

  return (
    <Container flex={{ justifyContent: 'space-between', alignItems: 'baseline' }}>
      <Container width={'63%'}>
        <GitspaceSelect
          loading={loading}
          formikName="code_repo_url"
          formInputClassName={css.repoAndBranch}
          text={<RepositoryText value={values.code_repo_url} repoList={repositories} isCDE={isCDE} />}
          tooltipProps={{
            onClose: () => {
              setRepoSearch('')
            }
          }}
          renderMenu={
            <Menu>
              {hideInitialMenu && (
                <Container margin={'small'}>
                  <ExpandingSearchInput
                    placeholder={getString('cde.create.searchRepositoryPlaceholder')}
                    alwaysExpanded
                    autoFocus={false}
                    defaultValue={repoSearch}
                    onChange={setRepoSearch}
                  />
                </Container>
              )}
              {loading ? (
                <MenuItem disabled text={getString('loading')} />
              ) : repoListOptions?.length ? (
                repoListOptions.map(repo => (
                  <MenuItem
                    key={repo.path}
                    text={
                      <Layout.Horizontal spacing="medium">
                        <img src={gitnessRepoLogo} height={16} width={16} />
                        <Text>{repo.path}</Text>
                      </Layout.Horizontal>
                    }
                    active={repo.git_url === values.code_repo_url}
                    onClick={() => {
                      const repoParams = repo?.path?.split('/') || []
                      formik.setValues((prvValues: any) => {
                        const codeRepoType = isCDE ? {} : { code_repo_type: EnumGitspaceCodeRepoType.GITNESS }
                        return {
                          ...prvValues,
                          code_repo_url: repo.git_url,
                          branch: repo.default_branch,
                          identifier: repoParams?.[repoParams.length - 1],
                          name: repo.path,
                          code_repo_ref: repo.path,
                          ...codeRepoType
                        }
                      })
                    }}
                  />
                ))
              ) : hideInitialMenu ? (
                <Container>
                  <NewRepoModalButton
                    space={space}
                    repoCreationType={RepoCreationType.CREATE}
                    customRenderer={fn => (
                      <MenuItem
                        icon="plus"
                        text={<String stringID="cde.create.repoNotFound" vars={{ repo: repoSearch }} useRichText />}
                        onClick={fn}
                      />
                    )}
                    modalTitle={getString('createRepo')}
                    onSubmit={() => {
                      refetchRepos()
                    }}
                  />
                </Container>
              ) : !hadReops ? (
                <Container>
                  <Layout.Vertical
                    spacing="medium"
                    className={css.noReposContainer}
                    flex={{ justifyContent: 'center' }}>
                    <img src={noRepo} height={90} width={90} />
                    <Layout.Vertical spacing="small" flex={{ alignItems: 'center' }}>
                      <Text color={Color.PRIMARY_10} font={{ size: 'normal', weight: 'bold' }}>
                        {getString('cde.getStarted')}
                      </Text>
                      <Text color={Color.PRIMARY_10} font={{ size: 'normal', weight: 'bold' }}>
                        {getString('cde.createImport')}
                      </Text>
                    </Layout.Vertical>
                    <NewRepoModalButton
                      space={space}
                      repoCreationType={RepoCreationType.CREATE}
                      customRenderer={fn => (
                        <Button width={'80%'} variation={ButtonVariation.PRIMARY} onClick={fn}>
                          {getString('createNewRepo')}
                        </Button>
                      )}
                      modalTitle={getString('newRepo')}
                      onSubmit={() => {
                        refetchRepos()
                      }}
                    />
                    <NewRepoModalButton
                      space={space}
                      repoCreationType={RepoCreationType.IMPORT}
                      customRenderer={fn => (
                        <Button width={'80%'} variation={ButtonVariation.SECONDARY} onClick={fn}>
                          {getString('cde.importInto')}
                        </Button>
                      )}
                      modalTitle={getString('importGitRepo')}
                      onSubmit={() => {
                        refetchRepos()
                      }}
                    />
                  </Layout.Vertical>
                </Container>
              ) : (
                <MenuItem disabled text={getString('loading')} />
              )}
            </Menu>
          }
          withoutCurrentColor
        />
      </Container>
      <Container width={'35%'}>
        <GitspaceSelect
          formikName="branch"
          loading={loadingBranches}
          disabled={!values.code_repo_url}
          formInputClassName={css.repoAndBranch}
          text={<BranchText value={values.branch} />}
          tooltipProps={{
            onClose: () => {
              setBranchSearch('')
            }
          }}
          renderMenu={
            <Menu>
              <Container margin={'small'}>
                <ExpandingSearchInput
                  placeholder={getString('cde.create.searchBranchPlaceholder')}
                  alwaysExpanded
                  autoFocus={false}
                  defaultValue={branchSearch}
                  onChange={setBranchSearch}
                />
              </Container>
              {loadingBranches ? (
                <MenuItem disabled text={getString('loading')} />
              ) : branches?.length ? (
                branches?.map(branch => (
                  <MenuItem
                    key={branch.name}
                    icon="git-branch"
                    text={branch.name}
                    active={branch.name === values.branch}
                    onClick={() => formik.setFieldValue('branch', branch.name)}
                  />
                ))
              ) : (
                <MenuItem
                  icon="warning-sign"
                  text={<String stringID="branchNotFound" vars={{ branch: branchSearch }} useRichText />}
                />
              )}
            </Menu>
          }
        />
      </Container>
    </Container>
  )
}
