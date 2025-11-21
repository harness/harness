import React from 'react'
import { DropDown, Layout, Button, SelectOption } from '@harnessio/uicore'
import { SortByTypes } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import type { EnumGitspaceSort } from 'services/cde'
import css from './SortByDropdown.module.scss'

interface SortByDropdownProps {
  value?: EnumGitspaceSort
  onChange?: (val: EnumGitspaceSort) => void
  onSortChange?: (val: EnumGitspaceSort) => void
  sortOrder?: 'asc' | 'desc'
  onOrderChange?: (val: 'asc' | 'desc') => void
}
export default function SortByDropdown(props: SortByDropdownProps): JSX.Element {
  const { value, onChange, onSortChange, sortOrder = 'asc', onOrderChange } = props
  const { getString } = useStrings()
  const dropdownList = SortByTypes(getString)

  const handleChange = (option: SelectOption) => {
    const newValue = option.value as EnumGitspaceSort
    if (onChange) {
      onChange(newValue)
    }
    if (onSortChange) {
      onSortChange(newValue)
    }
  }

  const handleOrderToggle = () => {
    if (onOrderChange) {
      onOrderChange(sortOrder === 'asc' ? 'desc' : 'asc')
    }
  }

  return (
    <Layout.Horizontal spacing="xsmall">
      <DropDown
        width={180}
        buttonTestId="gitspace-sort-select"
        items={dropdownList}
        value={value}
        onChange={option => handleChange(option)}
        placeholder={getString('cde.sortBy')}
      />
      {onOrderChange && (
        <Button
          icon={sortOrder === 'asc' ? 'sort-asc' : 'sort-desc'}
          className={css.sortOrderButton}
          minimal
          onClick={handleOrderToggle}
          disabled={!value}
        />
      )}
    </Layout.Horizontal>
  )
}
