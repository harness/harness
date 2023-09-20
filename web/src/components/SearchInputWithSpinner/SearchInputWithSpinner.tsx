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
  height?: number | string
  placeholder?: string
  icon?: IconName
  spinnerIcon?: IconName
  spinnerPosition?: 'left' | 'right'
  onSearch?: (searchTerm: string) => void
  onKeyDown?: (e: React.KeyboardEvent<HTMLElement>) => void
  readOnly?: boolean
  disabled?: boolean
  type?: string
}

export const SearchInputWithSpinner: React.FC<SearchInputWithSpinnerProps> = ({
  query = '',
  setQuery,
  loading = false,
  width = 250,
  height = 'var(--spacing-8)',
  placeholder,
  icon = 'search',
  spinnerIcon = 'steps-spinner',
  spinnerPosition = 'left',
  onSearch,
  onKeyDown,
  readOnly,
  disabled,
  type = 'search'
}) => {
  const { getString } = useStrings()
  const spinner = <Icon name={spinnerIcon as IconName} color={Color.PRIMARY_7} />
  const spinnerOnRight = spinnerPosition === 'right'

  return (
    <Container className={css.main}>
      <Layout.Horizontal className={css.layout}>
        <Render when={loading && !spinnerOnRight}>{spinner}</Render>
        <TextInput
          type={type}
          value={query}
          wrapperClassName={cx(css.wrapper, { [css.spinnerOnRight]: spinnerOnRight })}
          className={css.input}
          placeholder={placeholder || getString('search')}
          leftIcon={icon as IconName}
          style={{ width, height }}
          autoFocus={!readOnly && !disabled}
          onFocus={event => event.target.select()}
          onInput={event => {
            setQuery(event.currentTarget.value || '')
          }}
          onKeyDown={(e: React.KeyboardEvent<HTMLElement>) => {
            if (e.key === 'Enter') {
              onSearch?.((e as unknown as React.FormEvent<HTMLInputElement>).currentTarget.value || '')
            }
            onKeyDown?.(e)
          }}
          disabled={disabled}
          readOnly={readOnly}
        />
        <Render when={loading && spinnerOnRight}>{spinner}</Render>
      </Layout.Horizontal>
    </Container>
  )
}
