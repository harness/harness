import { useHistory } from 'react-router-dom'
import React, { useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, ButtonVariation, TextInput, Button } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, makeDiffRefs, PullRequestFilterOption } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
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
  const [filterOption, setFilterOption] = useState(activePullRequestFilterOption)
  const [searchTerm, setSearchTerm] = useState('')
  const items = useMemo(
    () => [
      { label: getString('open'), value: PullRequestFilterOption.OPEN },
      { label: getString('merged'), value: PullRequestFilterOption.MERGED },
      { label: getString('closed'), value: PullRequestFilterOption.CLOSED },
      { label: getString('rejected'), value: PullRequestFilterOption.REJECTED },
      // { label: getString('yours'), value: PullRequestFilterOption.YOURS },
      { label: getString('all'), value: PullRequestFilterOption.ALL }
    ],
    [getString]
  )
  const showSpinner = useMemo(() => loading, [loading])

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
        <FlexExpander />
        <TextInput
          className={css.input}
          placeholder={getString('search')}
          autoFocus
          onFocus={event => event.target.select()}
          value={searchTerm}
          onInput={event => {
            const value = event.currentTarget.value
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
          leftIcon={showSpinner ? CodeIcon.InputSpinner : CodeIcon.InputSearch}
        />
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
        />
      </Layout.Horizontal>
    </Container>
  )
}
