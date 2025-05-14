import React from 'react'
import { Button, Layout, Select, Text } from '@harnessio/uicore'
import cx from 'classnames'
import FormError from '../FormError/FormError'
import css from './CustomSelectDropdown.module.scss'

const CustomSelectDropdown = ({
  options,
  value,
  onChange,
  allowCustom = false,
  label = '',
  error = ''
}: {
  options?: any
  value: { label: string; value: string } | undefined
  onChange: any
  allowCustom?: boolean
  label: string
  error?: any
  placeholder?: string
}) => {
  return (
    <Layout.Vertical id="primary-borderless-buttons" className={css.mb15}>
      <Text className={css.customLabel}>{label}</Text>
      <Select
        items={options}
        value={value}
        onChange={onChange}
        className={cx(css.customSelect, error ? css.errorClass : '')}
        allowCreatingNewItems={allowCustom}
        itemRenderer={(item, props) => (
          <Button
            style={{ width: '100%', display: 'block' }}
            minimal
            onClick={ev => {
              props.handleClick(ev as any)
            }}>
            {item.label}
          </Button>
        )}></Select>
      <FormError message={error} />
    </Layout.Vertical>
  )
}

export default CustomSelectDropdown
