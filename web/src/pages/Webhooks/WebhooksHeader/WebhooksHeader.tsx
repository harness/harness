import { useHistory } from 'react-router-dom'
import React, { useState } from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Button } from '@harness/uicore'
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
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('createWebhook')}
          icon={CodeIcon.Add}
          onClick={() => history.push(routes.toCODEWebhookNew({ repoPath: repoMetadata?.path as string }))}
        />
        <FlexExpander />
        <SearchInputWithSpinner
          loading={loading}
          query={searchTerm}
          setQuery={value => {
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
        />
      </Layout.Horizontal>
    </Container>
  )
}
