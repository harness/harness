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
  const { value = [], onChange, disabled, orgIdentifiers } = props
  const { getString } = useStrings()
  const [searchQuery, setSearchQuery] = useState<string>('')

  const { projects, loading, hasMore, loadMore, search } = useProjects({
    searchTerm: searchQuery,
    orgIdentifiers
  })

  const handleSearch = useCallback(
    (query: string) => {
      setSearchQuery(query)
      search(query)
    },
    [search]
  )

  const filteredProjects = useMemo(() => {
    return projects
  }, [projects])

  useEffect(() => {
    if (orgIdentifiers?.length && value.length) {
      const validProjectIds = filteredProjects.map(project => project.identifier)
      const validValues = value.filter(id => validProjectIds.includes(id))

      if (validValues.length !== value.length) {
        onChange(validValues)
      }
    }
  }, [filteredProjects, value, onChange, orgIdentifiers])

  const allOptions = filteredProjects.map(project => ({
    value: project.identifier,
    label: project.name || project.identifier
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
            label: loading ? 'Loading' + '...' : 'Load More'
          }
        ]
      : [])
  ]

  const selectedOptions = allOptions.filter(option => value.includes(option.value))

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

      onChange(options.map(option => option.value as string))
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
