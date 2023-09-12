import React from 'react'
import { Render } from 'react-jsx-match'
import cx from 'classnames'
import { Container, Layout, TextInput } from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
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
  spinnerPosition?: 'left' | 'right'
  onSearch?: (searchTerm: string) => void
}

export const SearchInputWithSpinner: React.FC<SearchInputWithSpinnerProps> = ({
  query = '',
  setQuery,
  loading = false,
  width = 250,
  placeholder,
  icon = 'search',
  spinnerIcon = 'steps-spinner',
  spinnerPosition = 'left',
  onSearch
}) => {
  const { getString } = useStrings()
  const spinner = <Icon name={spinnerIcon as IconName} color={Color.PRIMARY_7} />
  const spinnerOnRight = spinnerPosition === 'right'

  return (
    <Container className={css.main}>
      <Layout.Horizontal className={css.layout}>
        <Render when={loading && !spinnerOnRight}>{spinner}</Render>
        <TextInput
          type="search"
          value={query}
          wrapperClassName={cx(css.wrapper, { [css.spinnerOnRight]: spinnerOnRight })}
          className={css.input}
          placeholder={placeholder || getString('search')}
          leftIcon={icon as IconName}
          style={{ width }}
          autoFocus
          onFocus={event => event.target.select()}
          onInput={event => {
            setQuery(event.currentTarget.value || '')
          }}
          onKeyDown={(e: React.KeyboardEvent<HTMLElement>) => {
            if (e.key === 'Enter') {
              onSearch?.((e as unknown as React.FormEvent<HTMLInputElement>).currentTarget.value || '')
            }
          }}
        />
        <Render when={loading && spinnerOnRight}>{spinner}</Render>
      </Layout.Horizontal>
    </Container>
  )
}
