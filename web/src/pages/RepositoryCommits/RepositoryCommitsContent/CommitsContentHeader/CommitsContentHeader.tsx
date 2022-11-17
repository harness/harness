import React, { ChangeEvent, useEffect, useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, Icon, Color, SelectOption } from '@harness/uicore'
import { uniq } from 'lodash-es'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import type { RepoBranch } from 'services/scm'
import { BRANCH_PER_PAGE } from 'utils/Utils'
import { GitIcon, GitInfoProps } from 'utils/GitUtils'
import css from './CommitsContentHeader.module.scss'

interface CommitsContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  onSwitch: (gitRef: string) => void
}

export function CommitsContentHeader({ repoMetadata, onSwitch }: CommitsContentHeaderProps) {
  const { getString } = useStrings()
  const [query, setQuery] = useState('')
  const [activeBranch, setActiveBranch] = useState(repoMetadata.defaultBranch)
  const { data, loading } = useGet<RepoBranch[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/branches`,
    queryParams: {
      sort: 'date',
      direction: 'desc',
      per_page: BRANCH_PER_PAGE,
      page: 1,
      query
    }
  })
  // defaultBranches is computed using repository default branch, and gitRef in URL, if it exists
  const defaultBranches = useMemo(() => [repoMetadata.defaultBranch].concat([]), [repoMetadata])
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
    <Container className={css.main}>
      <Layout.Horizontal spacing="medium">
        <DropDown
          icon={GitIcon.CodeBranch}
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
            onSwitch(switchBranch as string)
          }}
          popoverClassName={css.branchDropdown}
        />
        <FlexExpander />
      </Layout.Horizontal>
    </Container>
  )
}
