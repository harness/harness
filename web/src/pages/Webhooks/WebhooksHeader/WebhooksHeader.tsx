import { useHistory } from 'react-router-dom'
import React, { useState } from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Button } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import css from './WebhooksHeader.module.scss'

interface WebhooksHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  onSearchTermChanged: (searchTerm: string) => void
}

export function WebhooksHeader({ repoMetadata, loading, onSearchTermChanged }: WebhooksHeaderProps) {
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState('')
  const { routes } = useAppContext()
  const { getString } = useStrings()

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <SearchInputWithSpinner
          spinnerPosition="right"
          loading={loading}
          query={searchTerm}
          setQuery={value => {
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
        />
        <FlexExpander />
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('newWebhook')}
          icon={CodeIcon.Add}
          onClick={() => history.push(routes.toCODEWebhookNew({ repoPath: repoMetadata?.path as string }))}
        />
      </Layout.Horizontal>
    </Container>
  )
}
