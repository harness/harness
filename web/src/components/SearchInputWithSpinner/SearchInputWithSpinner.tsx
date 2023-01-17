import React from 'react'
import { Color, Container, Icon, IconName, Layout, TextInput } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './SearchInputWithSpinner.module.scss'

interface SearchInputWithSpinnerProps {
  query?: string
  setQuery: (value: string) => void
  loading?: boolean
  width?: number
  placeholder?: string
  icon?: IconName
  spinnerIcon?: IconName
}

export const SearchInputWithSpinner: React.FC<SearchInputWithSpinnerProps> = ({
  query = '',
  setQuery,
  loading = false,
  width = 250,
  placeholder,
  icon = 'search',
  spinnerIcon = 'spinner'
}) => {
  const { getString } = useStrings()
  return (
    <Container className={css.main}>
      <Layout.Horizontal className={css.layout}>
        {loading && <Icon name={spinnerIcon as IconName} color={Color.PRIMARY_7} />}
        <TextInput
          value={query}
          wrapperClassName={css.wrapper}
          className={css.input}
          placeholder={placeholder || getString('search')}
          leftIcon={icon as IconName}
          style={{ width }}
          autoFocus
          onFocus={event => event.target.select()}
          onInput={event => setQuery(event.currentTarget.value || '')}
        />
      </Layout.Horizontal>
    </Container>
  )
}
