import React, { useCallback, useState } from 'react'
import { Container, MultiSelectDropDown, type MultiSelectOption } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { useOrganizations } from 'cde-gitness/hooks/useOrganizations'
import { DROPDOWN_ACTIONS } from 'cde-gitness/pages/UsageDashboard/DropdownConstants'
import css from './OrganizationDropdown.module.scss'

export interface OrganizationDropdownProps {
  value: string[]
  onChange: (value: string[]) => void
  disabled?: boolean
  pageSize?: number
}

export default function OrganizationDropdown(props: OrganizationDropdownProps): JSX.Element {
  const { value = [], onChange, disabled } = props
  const { getString } = useStrings()
  const [searchQuery, setSearchQuery] = useState<string>('')

  const { organizations, loading, hasMore, loadMore, search } = useOrganizations({
    searchTerm: searchQuery
  })

  const handleSearch = useCallback(
    (query: string) => {
      setSearchQuery(query)
      search(query)
    },
    [search]
  )

  const safeOrganizations = Array.isArray(organizations) ? organizations : []

  const allOptions = safeOrganizations.map(org => ({
    value: org.identifier,
    label: org.name || org.identifier
  }))

  const items = [
    ...(value.length > 0
      ? [
          {
            value: DROPDOWN_ACTIONS.CLEAR_ALL,
            label: 'Clear All'
          }
        ]
      : []),
    ...allOptions,
    ...(hasMore
      ? [
          {
            value: DROPDOWN_ACTIONS.LOAD_MORE,
            label: loading ? 'Loading...' : 'Load More'
          }
        ]
      : [])
  ]

  const safeValue = Array.isArray(value) ? value : []
  const selectedOptions = allOptions.filter(option => safeValue.includes(option.value))

  const handleChange = useCallback(
    (options: MultiSelectOption[]) => {
      const clearAllOption = options.find(option => option.value === DROPDOWN_ACTIONS.CLEAR_ALL)
      if (clearAllOption) {
        onChange([])
        return
      }

      const loadMoreOption = options.find(option => option.value === DROPDOWN_ACTIONS.LOAD_MORE)
      if (loadMoreOption) {
        loadMore()
        const filteredOptions = options.filter(option => option.value !== DROPDOWN_ACTIONS.LOAD_MORE)
        onChange(filteredOptions.map(option => option.value as string))
        return
      }

      if (Array.isArray(options)) {
        onChange(options.map(option => option.value as string))
      }
    },
    [onChange, loadMore]
  )

  return (
    <Container className={css.customPopover}>
      <MultiSelectDropDown
        width={180}
        minWidth={180}
        buttonTestId="organization-select"
        placeholder={getString('cde.usageDashboard.organizations')}
        items={items as MultiSelectOption[]}
        value={selectedOptions as MultiSelectOption[]}
        onChange={handleChange}
        disabled={disabled}
        allowSearch={true}
        expandingSearchInputProps={{
          autoFocus: false,
          alwaysExpanded: true,
          defaultValue: searchQuery,
          onChange: handleSearch
        }}
      />
    </Container>
  )
}
