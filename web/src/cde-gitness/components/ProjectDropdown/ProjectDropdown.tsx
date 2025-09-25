import React, { useEffect, useMemo, useCallback, useState } from 'react'
import {
  Container,
  ExpandingSearchInput,
  Layout,
  Text,
  SortDropdown,
  sortByCreated,
  sortByLastModified,
  sortByName,
  SortMethod,
  Popover,
  Button,
  ButtonVariation,
  SelectOption,
  Select
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Position } from '@blueprintjs/core'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import { useProjects } from 'cde-gitness/hooks/useProjects'
import { useLocalStorage } from 'hooks/useLocalStorage'
import css from './ProjectDropdown.module.scss'

export interface ProjectDropdownProps {
  value: string[]
  onChange: (value: string[]) => void
  disabled?: boolean
  orgIdentifiers?: string[]
  pageSize?: number
}

interface ProjectItemProps {
  project: {
    identifier: string
    name: string
    orgIdentifier: string
    fullIdentifier: string
    organization?: {
      name: string
      identifier: string
    }
  }
  isSelected: boolean
  onToggle: () => void
}

const ProjectItem = ({ project, isSelected, onToggle }: ProjectItemProps): React.ReactElement => {
  const orgName = project.organization?.name || project.orgIdentifier

  return (
    <Container onClick={onToggle} className={cx(css.projectContainer, { [css.selected]: isSelected })}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.projectItemHorizontal}>
          <Container
            className={css.checkbox}
            border={{ color: isSelected ? Color.PRIMARY_7 : Color.GREY_300 }}
            background={isSelected ? Color.PRIMARY_7 : Color.WHITE}>
            {isSelected && <Icon name="tick" color={Color.WHITE} size={12} />}
          </Container>
          <Container
            className={cx(css.projectIcon, {
              [css.projectIconSelected]: isSelected,
              [css.projectIconUnselected]: !isSelected
            })}>
            <Icon name="nav-project" color={isSelected ? Color.WHITE : Color.TEAL_900} size={16} />
          </Container>
          <Text font={{ variation: FontVariation.BODY2 }} color={Color.GREY_900}>
            {project.name || project.identifier}
          </Text>
        </Layout.Horizontal>
        <Text className={css.orgInfoText} font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
          Organization: {orgName}
        </Text>
      </Container>
    </Container>
  )
}

export default function ProjectDropdown(props: ProjectDropdownProps): JSX.Element {
  const { value = [], onChange, disabled, orgIdentifiers = [] } = props
  const { getString } = useStrings()
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [isOpen, setIsOpen] = useState(false)
  const [selectedOrgFilter, setSelectedOrgFilter] = useState<string>('all')
  const [allLoadedProjects, setAllLoadedProjects] = useState<any[]>([])
  const LARGE_ORG_PAGE_SIZE = 1000
  const DEFAULT_PAGE_SIZE = 200

  const dynamicPageSize = useMemo(() => {
    if (selectedOrgFilter && selectedOrgFilter !== 'all') {
      return LARGE_ORG_PAGE_SIZE
    }
    return props.pageSize || DEFAULT_PAGE_SIZE
  }, [selectedOrgFilter, props.pageSize])

  const [sortPreference, setSortPreference] = useLocalStorage<SortMethod>(
    'sort-projectselector',
    SortMethod.LastModifiedDesc
  )

  const { projects, loading, hasMore, loadMore, search } = useProjects({
    searchTerm: searchQuery,
    pageSize: dynamicPageSize,
    orgIdentifier: selectedOrgFilter !== 'all' ? selectedOrgFilter : undefined,
    sortOrder: sortPreference
  })

  useEffect(() => {
    if (projects.length > 0) {
      setAllLoadedProjects(prevAll => {
        const existingIds = new Set(prevAll.map(p => p.fullIdentifier))
        const newProjects = projects.filter(p => !existingIds.has(p.fullIdentifier))
        return [...prevAll, ...newProjects]
      })
    }
  }, [projects])

  const handleSearch = useCallback(
    (query: string) => {
      setSearchQuery(query)
      search(query)
    },
    [search]
  )

  const validProjectIds = useMemo(
    () => new Set(allLoadedProjects.map(project => project.fullIdentifier)),
    [allLoadedProjects]
  )

  useEffect(() => {
    if (value.length && allLoadedProjects.length > 0) {
      const validValues = value.filter(id => validProjectIds.has(id))
      if (validValues.length !== value.length) {
        const hasSearched = searchQuery.length > 0 || selectedOrgFilter !== 'all'
        if (hasSearched || allLoadedProjects.length >= 200) {
          onChange(validValues)
        }
      }
    }
  }, [allLoadedProjects, value, onChange, searchQuery, selectedOrgFilter, validProjectIds])

  const orgFilterOptions: SelectOption[] = useMemo(() => {
    const options: SelectOption[] = [{ label: getString('cde.usageDashboard.allProjects'), value: 'all' }]

    const orgMap = new Map<string, string>()
    projects.forEach(project => {
      if (project.organization) {
        orgMap.set(project.orgIdentifier, project.organization.name || project.orgIdentifier)
      }
    })

    const orgOptions = orgIdentifiers.map(orgId => ({
      label: orgMap.get(orgId) || orgId,
      value: orgId
    }))

    options.push(...orgOptions)
    return options
  }, [projects, orgIdentifiers, getString])

  const hasMoreFiltered = useMemo(() => {
    if (selectedOrgFilter && selectedOrgFilter !== 'all') {
      return false
    }

    return hasMore
  }, [hasMore, selectedOrgFilter])

  const buttonText = getString('cde.usageDashboard.projects')
  const selectedCount = value.length

  const dropdownContent = (
    <Container className={css.dropdownContent}>
      <Layout.Horizontal spacing={'small'} className={css.searchContainer}>
        <ExpandingSearchInput
          onChange={handleSearch}
          className={css.searchWidth}
          alwaysExpanded
          autoFocus={false}
          placeholder={getString('cde.usageDashboard.searchProjectPlaceholder')}
        />
        <Container className={css.orgFilterWidth}>
          <Select
            items={orgFilterOptions}
            value={orgFilterOptions.find(opt => opt.value === selectedOrgFilter)}
            onChange={option => setSelectedOrgFilter(option.value as string)}
          />
        </Container>
        <Container className={css.sortWidth}>
          <SortDropdown
            selectedSortMethod={sortPreference}
            onSortMethodChange={option => {
              if (option) {
                setSortPreference(option.value as SortMethod)
              }
            }}
            sortOptions={[...sortByLastModified, ...sortByCreated, ...sortByName]}
          />
        </Container>
      </Layout.Horizontal>

      {loading ? (
        <Container className={css.loadingContainer}>
          <Icon name="loading" size={24} color={Color.PRIMARY_7} />
        </Container>
      ) : null}

      {!loading && projects.length === 0 ? (
        <Container height={120} className={css.emptyContainer}>
          <Text font={{ variation: FontVariation.H5 }} className={css.noResultsTitle}>
            {getString('cde.usageDashboard.noResultsFound')}
          </Text>
          <Text color={Color.GREY_500}>{getString('cde.usageDashboard.tryDifferentKeywords')}</Text>
        </Container>
      ) : !loading ? (
        <Container className={css.projectsContainer}>
          {value.length > 0 && (
            <Container
              onClick={() => {
                onChange([])
                setSearchQuery('')
                setSelectedOrgFilter('all')
                search('')
              }}
              className={css.actionContainer}>
              <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                <Icon name="main-trash" size={16} color={Color.RED_500} />
                <Text font={{ variation: FontVariation.BODY2 }} color={Color.RED_500}>
                  {getString('cde.usageDashboard.clearAll')}
                </Text>
              </Layout.Horizontal>
            </Container>
          )}

          {projects.map(project => {
            const isSelected = value.includes(project.fullIdentifier)
            return (
              <ProjectItem
                key={project.fullIdentifier}
                project={project}
                isSelected={isSelected}
                onToggle={() => {
                  if (isSelected) {
                    onChange(value.filter(id => id !== project.fullIdentifier))
                  } else {
                    onChange([...value, project.fullIdentifier])
                  }
                }}
              />
            )
          })}

          {hasMoreFiltered && (
            <Container onClick={() => loadMore()} className={cx(css.actionContainer)}>
              <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                <Icon name={loading ? 'loading' : 'chevron-down'} size={16} color={Color.PRIMARY_7} />
                <Text font={{ variation: FontVariation.BODY2 }} color={Color.PRIMARY_7}>
                  {loading ? getString('cde.usageDashboard.loading') : getString('cde.usageDashboard.loadMore')}
                </Text>
              </Layout.Horizontal>
            </Container>
          )}
        </Container>
      ) : null}
    </Container>
  )

  return (
    <Container className={css.customPopover}>
      <Popover
        isOpen={isOpen}
        onInteraction={(nextOpenState, _event) => {
          setIsOpen(nextOpenState)
        }}
        position={Position.BOTTOM_LEFT}
        className={css.popover}
        popoverClassName={css.popoverContent}
        content={dropdownContent}>
        <Button
          variation={ButtonVariation.TERTIARY}
          rightIcon="main-chevron-down"
          disabled={disabled}
          className={css.dropdownButton}
          data-testid="project-select"
          onClick={() => setIsOpen(!isOpen)}
          iconProps={{
            size: 8,
            color: 'var(--grey-400)'
          }}>
          <Layout.Horizontal spacing="small" className={css.buttonContent}>
            <Text className={css.buttonText}>{buttonText}</Text>
            {selectedCount > 0 && (
              <Container className={css.countBadge}>
                <Text className={css.countText}>{selectedCount < 10 ? `0${selectedCount}` : selectedCount}</Text>
              </Container>
            )}
          </Layout.Horizontal>
        </Button>
      </Popover>
    </Container>
  )
}
