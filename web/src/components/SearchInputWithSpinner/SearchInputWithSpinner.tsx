import React from 'react'
import { Render } from 'react-jsx-match'
import cx from 'classnames'
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
  spinnerPosition?: 'left' | 'right'
}

export const SearchInputWithSpinner: React.FC<SearchInputWithSpinnerProps> = ({
  query = '',
  setQuery,
  loading = false,
  width = 250,
  placeholder,
  icon = 'search',
  spinnerIcon = 'spinner',
  spinnerPosition = 'left'
}) => {
  const { getString } = useStrings()
  const spinner = <Icon name={spinnerIcon as IconName} color={Color.PRIMARY_7} />
  const spinnerOnRight = spinnerPosition === 'right'

  return (
    <Container className={css.main}>
      <Layout.Horizontal className={css.layout}>
        <Render when={loading && !spinnerOnRight}>{spinner}</Render>
        <TextInput
          value={query}
          wrapperClassName={cx(css.wrapper, { [css.spinnerOnRight]: spinnerOnRight })}
          className={css.input}
          placeholder={placeholder || getString('search')}
          leftIcon={icon as IconName}
          style={{ width }}
          autoFocus
          onFocus={event => event.target.select()}
          onInput={event => setQuery(event.currentTarget.value || '')}
        />
        <Render when={loading && spinnerOnRight}>{spinner}</Render>
      </Layout.Horizontal>
    </Container>
  )
}
