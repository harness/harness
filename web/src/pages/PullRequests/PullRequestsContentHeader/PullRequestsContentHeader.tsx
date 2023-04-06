import { useHistory } from 'react-router-dom'
import React, { useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, ButtonVariation, Button } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, makeDiffRefs, PullRequestFilterOption } from 'utils/GitUtils'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { permissionProps } from 'utils/Utils'
import css from './PullRequestsContentHeader.module.scss'

interface PullRequestsContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  activePullRequestFilterOption?: string
  onPullRequestFilterChanged: (filter: string) => void
  onSearchTermChanged: (searchTerm: string) => void
}

export function PullRequestsContentHeader({
  loading,
  onPullRequestFilterChanged,
  onSearchTermChanged,
  activePullRequestFilterOption = PullRequestFilterOption.OPEN,
  repoMetadata
}: PullRequestsContentHeaderProps) {
  const history = useHistory()
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const [filterOption, setFilterOption] = useUserPreference(
    UserPreference.PULL_REQUESTS_FILTER_SELECTED_OPTIONS,
    activePullRequestFilterOption
  )
  const [searchTerm, setSearchTerm] = useState('')
  const space = useGetSpaceParam()

  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPO'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const items = useMemo(
    () => [
      { label: getString('open'), value: PullRequestFilterOption.OPEN },
      { label: getString('merged'), value: PullRequestFilterOption.MERGED },
      { label: getString('closed'), value: PullRequestFilterOption.CLOSED },
      { label: getString('draft'), value: PullRequestFilterOption.DRAFT },
      // { label: getString('yours'), value: PullRequestFilterOption.YOURS },
      { label: getString('all'), value: PullRequestFilterOption.ALL }
    ],
    [getString]
  )

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <DropDown
          value={filterOption}
          items={items}
          onChange={({ value }) => {
            setFilterOption(value as string)
            onPullRequestFilterChanged(value as string)
          }}
          popoverClassName={css.branchDropdown}
        />
        <SearchInputWithSpinner
          loading={loading}
          query={searchTerm}
          setQuery={value => {
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
          spinnerPosition="right"
        />
        <FlexExpander />
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('newPullRequest')}
          icon={CodeIcon.Add}
          onClick={() => {
            history.push(
              routes.toCODECompare({
                repoPath: repoMetadata?.path as string,
                diffRefs: makeDiffRefs(repoMetadata?.default_branch as string, '')
              })
            )
          }}
          {...permissionProps(permPushResult, standalone)}
        />
      </Layout.Horizontal>
    </Container>
  )
}
