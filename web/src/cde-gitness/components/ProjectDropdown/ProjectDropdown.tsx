import React, { useEffect, useMemo, useCallback, useState } from 'react'
import { Container, MultiSelectDropDown, type MultiSelectOption } from '@harnessio/uicore'

import { useStrings } from 'framework/strings'
import { useProjects } from 'cde-gitness/hooks/useProjects'
import { DROPDOWN_ACTIONS } from 'cde-gitness/pages/UsageDashboard/DropdownConstants'
import css from './ProjectDropdown.module.scss'

export interface ProjectDropdownProps {
  value: string[]
  onChange: (value: string[]) => void
  disabled?: boolean
  orgIdentifiers?: string[]
  pageSize?: number
}

export default function ProjectDropdown(props: ProjectDropdownProps): JSX.Element {
  const { value = [], onChange, disabled } = props
  const { getString } = useStrings()
  const [searchQuery, setSearchQuery] = useState<string>('')

  const effectivePageSize = props.pageSize || 200

  const { projects, loading, hasMore, loadMore, search } = useProjects({
    searchTerm: searchQuery,
    pageSize: effectivePageSize
  })

  const handleSearch = useCallback(
    (query: string) => {
      setSearchQuery(query)
      search(query)
    },
    [search]
  )

  useEffect(() => {
    if (value.length && projects.length > 0) {
      const validProjectIds = projects.map(project => project.fullIdentifier)
      const validValues = value.filter(id => validProjectIds.includes(id))

      if (validValues.length !== value.length) {
        onChange(validValues)
      }
    }
  }, [projects, value, onChange])

  const allOptions = projects.map(project => ({
    value: project.fullIdentifier,
    label: project.name || project.identifier
  }))

  const selectedOptions = value.map(selectedId => {
    const existingOption = allOptions.find(option => option.value === selectedId)
    if (existingOption) return existingOption

    return {
      value: selectedId,
      label: selectedId
    }
  })

  const showLoadMore = hasMore

  const items = useMemo(() => {
    const clearAllItem =
      value.length > 0
        ? [
            {
              value: DROPDOWN_ACTIONS.CLEAR_ALL,
              label: 'Clear All'
            }
          ]
        : []

    const loadMoreItem = showLoadMore
      ? [
          {
            value: DROPDOWN_ACTIONS.LOAD_MORE,
            label: loading ? 'Loading' + '...' : 'Load More'
          }
        ]
      : []

    return [...clearAllItem, ...allOptions, ...loadMoreItem]
  }, [value, allOptions, showLoadMore, loading])

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

      const newValues = options.map(option => option.value as string)
      onChange(newValues)
    },
    [onChange, loadMore]
  )

  return (
    <Container className={css.customPopover}>
      <MultiSelectDropDown
        width={180}
        minWidth={180}
        placeholder={getString('cde.usageDashboard.projects')}
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
