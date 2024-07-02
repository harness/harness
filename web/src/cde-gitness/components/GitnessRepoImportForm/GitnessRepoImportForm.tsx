import React, { useEffect, useState } from 'react'
import { useGet } from 'restful-react'
import { Container, ExpandingSearchInput, Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { useFormikContext } from 'formik'
import type { TypesRepository } from 'services/code'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { String, useStrings } from 'framework/strings'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import NewRepoModalButton from 'components/NewRepoModalButton/NewRepoModalButton'
import { GitspaceSelect } from '../../../cde/components/GitspaceSelect/GitspaceSelect'
import gitnessRepoLogo from './gitness.svg?url'
import css from './GitnessRepoImportForm.module.scss'

const RepositoryText = ({ repoList, value }: { repoList: TypesRepository[] | null; value?: string }) => {
  const { getString } = useStrings()
  const repoMetadata = repoList?.find(repo => repo.git_url === value)
  const repoName = repoMetadata?.path

  return (
    <Layout.Horizontal spacing={'medium'} flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
      <img src={gitnessRepoLogo} height={24} width={24} />
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

export const GitnessRepoImportForm = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const [branchSearch, setBranchSearch] = useState('')
  const [repoSearch, setRepoSearch] = useState('')
  const [repoRef, setReporef] = useState('')

  const {
    data: repositories,
    loading,
    refetch: refetchRepos
  } = useGet<TypesRepository[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    queryParams: { query: repoSearch },
    debounce: 500
  })

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
          text={<RepositoryText value={values.code_repo_url} repoList={repositories} />}
          tooltipProps={{
            onClose: () => {
              setRepoSearch('')
            }
          }}
          renderMenu={
            <Menu>
              <Container margin={'small'}>
                <ExpandingSearchInput
                  placeholder={getString('cde.create.searchRepositoryPlaceholder')}
                  alwaysExpanded
                  autoFocus={false}
                  defaultValue={repoSearch}
                  onChange={setRepoSearch}
                />
              </Container>
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
                      formik.setValues((prvValues: any) => {
                        return {
                          ...prvValues,
                          code_repo_url: repo.git_url,
                          id: repo.path,
                          name: repo.path
                        }
                      })
                      formik.setFieldValue('code_repo_url', repo.git_url)
                    }}
                  />
                ))
              ) : (
                <Container>
                  <NewRepoModalButton
                    space={space}
                    newRepoModalOnly
                    notFoundRepoName={repoSearch}
                    modalTitle={getString('createRepo')}
                    onSubmit={() => {
                      refetchRepos()
                    }}
                  />
                </Container>
              )}
            </Menu>
          }
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
