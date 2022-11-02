import React, { ChangeEvent, useEffect, useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, Icon, Color, SelectOption } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { uniq } from 'lodash-es'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { RepoBranch, TypesRepository } from 'services/scm'
import { BRANCH_PER_PAGE } from 'utils/Utils'
import { GitIcon } from 'utils/GitUtils'
import css from './CommitsContentHeader.module.scss'

interface CommitsContentHeaderProps {
  gitRef?: string
  resourcePath?: string
  repoMetadata: TypesRepository
}

export function CommitsContentHeader({
  repoMetadata,
  gitRef,
  resourcePath = ''
}: CommitsContentHeaderProps): JSX.Element {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()
  const [query, setQuery] = useState('')
  const [activeBranch, setActiveBranch] = useState(gitRef || repoMetadata.defaultBranch)
  const path = useMemo(
    () =>
      `/api/v1/repos/${repoMetadata.path}/+/branches?sort=date&direction=desc&per_page=${BRANCH_PER_PAGE}&page=1&query=${query}`,
    [query, repoMetadata.path]
  )
  const { data, loading } = useGet<RepoBranch[]>({ path })
  // defaultBranches is computed using repository default branch, and gitRef in URL, if it exists
  const defaultBranches = useMemo(
    () => [repoMetadata.defaultBranch].concat(gitRef ? gitRef : []),
    [repoMetadata, gitRef]
  )
  const [branches, setBranches] = useState<SelectOption[]>(
    uniq(defaultBranches.map(_branch => ({ label: _branch, value: _branch }))) as SelectOption[]
  )

  useEffect(() => {
    if (data?.length) {
      setBranches(
        uniq(defaultBranches.concat(data.map(e => e.name) as string[])).map(_branch => ({
          label: _branch,
          value: _branch
        })) as SelectOption[]
      )
    }
  }, [data, defaultBranches])

  return (
    <Container className={css.folderHeader}>
      <Layout.Horizontal spacing="medium">
        <DropDown
          icon={GitIcon.BRANCH}
          value={activeBranch}
          items={branches}
          {...{
            inputProps: {
              leftElement: (
                <Icon name={loading ? 'steps-spinner' : 'thinner-search'} size={12} color={Color.GREY_500} />
              ),
              placeholder: getString('search'),
              onInput: (event: ChangeEvent<HTMLInputElement>) => {
                if (event.target.value !== query) {
                  setQuery(event.target.value)
                }
              },
              onBlur: (event: ChangeEvent<HTMLInputElement>) => {
                setTimeout(() => {
                  setQuery(event.target.value || '')
                }, 250)
              }
            }
          }}
          onChange={({ value: switchBranch }) => {
            setActiveBranch(switchBranch as string)
            history.push(
              routes.toSCMRepository({
                repoPath: repoMetadata.path as string,
                gitRef: switchBranch as string,
                resourcePath // TODO: Handle 404 when resourcePath does not exist in newly switched branch
              })
            )
          }}
          popoverClassName={css.branchDropdown}
        />
        <FlexExpander />
      </Layout.Horizontal>
    </Container>
  )
}

// TODO: Optimize branch fetching when first fetch return less than request PER_PAGE
